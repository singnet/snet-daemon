package blockchain

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/ipfsutils"
	log "github.com/sirupsen/logrus"
	"math/big"
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
      "group_name": "default_group2",
      "group_id": "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
      "payment": {
        "payment_address": "0x671276c61943A35D5F230d076bDFd91B0c47bF09",
        "payment_expiration_threshold": 40320,
        "payment_channel_storage_type": "etcd",
        "payment_channel_storage_client": {
          "connection_timeout": "5s",
          "request_timeout": "3s",
          "endpoints": [
            "http://127.0.0.1:2379"
          ]
        }
      }
    },

    {
      "group_name": "default_group2",
      "group_id": "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
      "payment": {
        "payment_address": "0x671276c61943A35D5F230d076bDFd91B0c47bF09",
        "payment_expiration_threshold": 40320,
        "payment_channel_storage_type": "etcd",
        "payment_channel_storage_client": {
          "connection_timeout": "5s",
          "request_timeout": "3s",
          "endpoints": [
            "http://127.0.0.1:2379"
          ]
        }
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
	daemonGroupID           [32]byte
	recipientPaymentAddress common.Address
}

type Payment struct {
	PaymentAddress              string                      `json:"payment_address"`
	PaymentExpirationThreshold  *big.Int                    `json:"payment_expiration_threshold"`
	PaymentChannelStorageType   string                      `json:"payment_channel_storage_type"`
	PaymentChannelStorageClient PaymentChannelStorageClient `json:"payment_channel_storage_client"`
}

//Structure to hold the individual group details , an Organization can have multiple groups
type Group struct {
	GroupName      string  `json:"group_name"`
	GroupID        string  `json:"group_id"`
	PaymentDetails Payment `json:"payment"`
}

//Structure to hold the storage details of the payment
type PaymentChannelStorageClient struct {
	ConnectionTimeout string   `json:"connection_timeout"`
	RequestTimeout    string   `json:"request_timeout"`
	Endpoints         []string `json:"endpoints"`
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
		log.WithError(err)
		return nil, err
	}

	return metaData, nil
}

func setDerivedAttributes(metaData *OrganizationMetaData) (err error) {
	if metaData.daemonGroup, err = getDaemonGroup(*metaData); err != nil {
		return err
	}
	metaData.daemonGroupID, err = ConvertBase64Encoding(metaData.daemonGroup.GroupID)
	metaData.recipientPaymentAddress = common.HexToAddress(metaData.daemonGroup.PaymentDetails.PaymentAddress)
	return err
}

//Determine the group this Daemon belongs to
func getDaemonGroup(metaData OrganizationMetaData) (group *Group, err error) {
	groupName := config.GetString(config.DaemonGroupName)
	for _, group := range metaData.Groups {
		if strings.Compare(group.GroupName, groupName) == 0 {
			return &group, nil
		}
	}
	err = fmt.Errorf("group name %v in config is invalid, "+
		"there was no group found with this name in the metadata", groupName)
	log.WithError(err)
	return nil, err
}

//Will be used to load the Organization metadata when Daemon starts
//To be part of components
func GetOrganizationMetaData() *OrganizationMetaData {
	var metadata *OrganizationMetaData
	var err error
	if config.GetBool(config.BlockchainEnabledKey) {
		ipfsHash := string(getMetaDataURI())
		metadata, err = GetOrganizationMetaDataFromIPFS(FormatHash(ipfsHash))
	}
	if err != nil {
		log.WithError(err).
			Panic("error on retrieving / parsing organization metadata from block chain")
	}
	return metadata
}

func GetOrganizationMetaDataFromIPFS(hash string) (*OrganizationMetaData, error) {
	jsondata := ipfsutils.GetIpfsFile(hash)
	return InitOrganizationMetaDataFromJson(jsondata)
}

//TODO , once the latest contract is pushed , the below method will be called
func getMetaDataURI() []byte {
	//Block chain call here to get the hash of the metadata for the given Organization
	reg := getRegistryCaller()
	orgId := StringToBytes32(config.GetString(config.OrganizationId))

	organizationRegistered, err := reg.GetOrganizationById(nil, orgId)
	if err != nil || !organizationRegistered.Found {
		log.WithError(err).WithField("OrganizationId", config.GetString(config.OrganizationId)).
			WithField("ServiceId", config.GetString(config.ServiceId)).
			Panic("Error Retrieving contract details for the Given Organization and Service Ids ")
	}
	//return organizationRegistered.metaDataURI[:] //TODO , once the latest version of the registry is published, this line will be uncommented.
	return nil
}

//Get the Group ID the Daemon needs to associate itself to , requests belonging to a different group if will be rejected
func (metaData OrganizationMetaData) GetGroupIdString() string {
	return metaData.daemonGroup.GroupID
}

// Return the group id in bytes
func (metaData OrganizationMetaData) GetGroupId() [32]byte {
	return metaData.daemonGroupID
}

//Pass the group Name and retrieve the details of the payment address/ recipient address.
func (metaData OrganizationMetaData) GetPaymentAddress() common.Address {
	return metaData.recipientPaymentAddress
}

//Payment expiration threshold
func (metaData *OrganizationMetaData) GetPaymentExpirationThreshold() *big.Int {
	return metaData.daemonGroup.PaymentDetails.PaymentExpirationThreshold
}

//Get the End points of the Payment Storage used to update the storage state
func (metaData OrganizationMetaData) GetPaymentStorageEndPoint() []string {
	return metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.Endpoints
}
