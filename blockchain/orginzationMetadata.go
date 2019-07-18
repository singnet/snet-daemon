package blockchain

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/ipfsutils"
	log "github.com/sirupsen/logrus"
	"strings"
)

/*
This metadata structure defines the organization to group mappings.
Please note , that groups are logical bucketing of managing payments ( same recipient for each group across the services in an organization)
A given Organization will be associated with multiple groups and every group will be associated to a payment
Sample example of the JSON structure from the block chain is given below .
Please note that all the services that belong to a given group in an organization will have the same recipient address.
*/

/*

{
"org_name": "organization_name",
"org_id": "org_id1",
"groups": [
		{
			"group_name": "group1",
			"group_id": "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
			"payment_address": "0x671276c61943A35D5F230d076bDFd91B0c47bF09",
			"payment_channel_storage_type": "etcd",
			"payment_channel_storage_client": {
			"connection_timeout": "5s",
			"request_timeout": "3s",
			"endpoints": [
			"http://127.0.0.1:2379"
			]
			}
		},
		{
			"group_name": "default_group",
			"group_id": "88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
			"payment_address": "0x671276c61943A35D5F230d076bDFd91B0c47bF09",
			"payment_channel_storage_type": "etcd",
			"payment_channel_storage_client": {
			"connection_timeout": "5s",
			"request_timeout": "3s",
			"endpoints": [
			"http://127.0.0.1:2479"
			]
		}
		}
]
}*/

type OrganizationMetaData struct {
	OrgName string  `json:"org_name"`
	OrgID   string  `json:"org_id"`
	Groups  []Group `json:"groups"`
	//This will used to determine which group the daemon belongs to
	daemonGroup             *Group
	daemonReplicaGroupID    [32]byte
	recipientPaymentAddress common.Address
}

//Structure to hold the individual group details , an Organization can have multiple groups
type Group struct {
	GroupName                   string                      `json:"group_name"`
	GroupID                     string                      `json:"group_id"`
	PaymentAddress              string                      `json:"payment_address"`
	PaymentChannelStorageType   string                      `json:"payment_channel_storage_type"`
	PaymentChannelStorageClient PaymentChannelStorageClient `json:"payment_channel_storage_client"`
}

//Structure to hold the storage details of the payment
type PaymentChannelStorageClient struct {
	ConnectionTimeout string   `json:"connection_timeout"`
	RequestTimeout    string   `json:"request_timeout"`
	Endpoints         []string `json:"endpoints"`
}

//Get the Group ID the Daemon needs to associate itself to , requests belonging to a different group if will be rejected
func (orgMetaData OrganizationMetaData) GetGroupId() (groupId string, err error) {
	if group, err := orgMetaData.getDaemonGroup(); err == nil {
		return group.GroupID, nil
	}
	return groupId, err
}

//Construct the Organization metadata from the JSON Passed
func InitOrganizationMetaDataFromJson(jsonData string) (metaData *OrganizationMetaData, err error) {
	metaData = new(OrganizationMetaData)
	err = json.Unmarshal([]byte(jsonData), &metaData)
	if err != nil {
		log.WithError(err).WithField("jsondata", jsonData)
		return nil, err
	}

	if err = setDerivedAttributes(metaData); err != nil {
		return nil, err
	}

	return metaData, err
}

func setDerivedAttributes(metaData *OrganizationMetaData) (err error) {
	if metaData.daemonGroup, err = metaData.getDaemonGroup(); err != nil {
		return err
	}
	metaData.daemonReplicaGroupID, err = ConvertBase64Encoding(metaData.daemonGroup.GroupID)
	metaData.recipientPaymentAddress = common.HexToAddress(metaData.daemonGroup.PaymentAddress)
	return err
}

//Pass the group Name and retrieve the details of the payment address/ recipient address.
func (orgMetaData OrganizationMetaData) GetPaymentAddress() (address string) {
	return orgMetaData.daemonGroup.PaymentAddress
}

func (orgMetaData OrganizationMetaData) getDaemonGroup() (group *Group, err error) {
	groupName := config.GetString(config.DaemonGroupName)
	for _, group := range orgMetaData.Groups {
		if strings.Compare(group.GroupName, groupName) == 0 {
			return &group, nil
		}
	}
	return nil, fmt.Errorf("group Name %v in config is invalid, "+
		"there was no group found with this name in the metadata", groupName)
}

//Get the End points of the Payment Storage used to update the storage state
func (orgMetaData OrganizationMetaData) GetPaymentStorageEndPoint() (endpoint []string) {
	return orgMetaData.daemonGroup.PaymentChannelStorageClient.Endpoints
}

func (metaData OrganizationMetaData) GetDaemonGroupID() [32]byte {
	return metaData.daemonReplicaGroupID
}

//Will be used to load the Organization metadata when Daemon starts
//To be part of components
func GetOrganizationMetaData() *OrganizationMetaData {
	var metadata *OrganizationMetaData
	var err error
	if config.GetBool(config.BlockchainEnabledKey) {
		ipfsHash := string(getOrgMetadataURI())
		metadata, err = GetOrganizationMetaDataFromIPFS(FormatHash(ipfsHash))
	}
	if err != nil {
		log.WithError(err).
			Panic("error on determining organization metadata from block chain")
	}
	return metadata
}

func GetOrganizationMetaDataFromIPFS(hash string) (*OrganizationMetaData, error) {
	jsondata := ipfsutils.GetIpfsFile(hash)
	return InitOrganizationMetaDataFromJson(jsondata)
}

//TODO , once the latest contract is pushed , the below method will be called
func getOrgMetadataURI() []byte {
	//Block chain call here to get the hash of the metadata for the given Organization
	reg := getRegistryCaller()
	orgId := StringToBytes32(config.GetString(config.OrganizationId))

	organizationRegistered, err := reg.GetOrganizationById(nil, orgId)
	if err != nil || !organizationRegistered.Found {
		log.WithError(err).WithField("OrganizationId", config.GetString(config.OrganizationId)).
			WithField("ServiceId", config.GetString(config.ServiceId)).
			Panic("Error Retrieving contract details for the Given Organization and Service Ids ")
	}

	//return organizationRegistered.orgMetadataURI[:] //TODO , once the latest version of the registry is published, this line will be uncommented.
	return nil
}
