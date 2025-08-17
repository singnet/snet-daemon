package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/ipfsutils"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

/*
This metadata structure defines the organization to group mappings.
Please note that groups are logical bucketing of managing payments (the same recipient for each group across the services in an organization)
A given Organization will be associated with multiple groups, and every group will be associated to a payment
Sample example of the JSON structure from the blockchain is given below.
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
      "license_server_endpoints": [
        "http://licenseendpont:8082"
      ],
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

// Group Structure to hold the individual group details, an Organization can have multiple groups
type Group struct {
	GroupName        string   `json:"group_name"`
	GroupID          string   `json:"group_id"`
	PaymentDetails   Payment  `json:"payment"`
	LicenseEndpoints []string `json:"license_server_endpoints"`
}

// PaymentChannelStorageClient to hold the storage details of the payment
type PaymentChannelStorageClient struct {
	ConnectionTimeout string   `json:"connection_timeout" mapstructure:"connection_timeout"`
	RequestTimeout    string   `json:"request_timeout" mapstructure:"request_timeout"`
	Endpoints         []string `json:"endpoints"`
}

// InitOrganizationMetaDataFromJson Construct the Organization metadata from the JSON Passed
func InitOrganizationMetaDataFromJson(jsonData []byte) (metaData *OrganizationMetaData, err error) {
	metaData = new(OrganizationMetaData)
	err = json.Unmarshal(jsonData, &metaData)
	if err != nil {
		zap.L().Error("Error in unmarshalling metadata json", zap.Error(err), zap.Any("jsondata", jsonData))
		return nil, err
	}

	// Check for mandatory validations
	if err = setDerivedAttributes(metaData); err != nil {
		zap.L().Error("Error in setting derived attributes", zap.Error(err))
		return nil, err
	}
	if err = checkMandatoryFields(metaData); err != nil {
		zap.L().Error("Error in check mandatory fields", zap.Error(err))
		return nil, err
	}

	return metaData, nil
}

func checkMandatoryFields(metaData *OrganizationMetaData) (err error) {
	if metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.Endpoints == nil {
		err = fmt.Errorf("Mandatory field : ETCD Client Endpoints are mising for the Group %v ", metaData.daemonGroup.GroupName)
	}

	if metaData.recipientPaymentAddress == (common.Address{}) {
		err = fmt.Errorf("Mandatory field : Recipient Address is missing for the Group %v ", metaData.daemonGroup.GroupName)
	}
	return err
}

func setDerivedAttributes(metaData *OrganizationMetaData) (err error) {
	if metaData.daemonGroup, err = getDaemonGroup(*metaData); err != nil {
		return err
	}
	metaData.daemonGroupID, err = utils.ConvertBase64Encoding(metaData.daemonGroup.GroupID)
	//use the checksum address (convert the address in to a checksum address and set it back)
	metaData.daemonGroup.PaymentDetails.PaymentAddress = utils.ToChecksumAddressStr(metaData.daemonGroup.PaymentDetails.PaymentAddress)
	metaData.recipientPaymentAddress = common.HexToAddress(metaData.daemonGroup.PaymentDetails.PaymentAddress)

	return err
}

// Determine the group this Daemon belongs to
func getDaemonGroup(metaData OrganizationMetaData) (group *Group, err error) {
	groupName := config.GetString(config.DaemonGroupName)
	for _, group := range metaData.Groups {
		if strings.Compare(group.GroupName, groupName) == 0 {
			return &group, nil
		}
	}
	err = fmt.Errorf("group name %v in config is invalid, "+
		"there was no group found with this name in the metadata", groupName)
	zap.L().Error("error in getting daemon group", zap.Error(err))
	return nil, err
}

// GetOrganizationMetaData will be used to load the Organization metadata when Daemon starts
// To be part of components
func GetOrganizationMetaData() *OrganizationMetaData {
	var metadata *OrganizationMetaData
	var err error
	if config.GetBool(config.BlockchainEnabledKey) {
		ipfsHash := string(getMetaDataURI())
		metadata, err = GetOrganizationMetaDataFromIPFS(ipfsHash)
	} else {
		metadata = &OrganizationMetaData{daemonGroup: &Group{}}
	}
	if err != nil {
		zap.L().Panic("error on retrieving / parsing organization metadata from block chain", zap.Error(err))
	}
	return metadata
}

func GetOrganizationMetaDataFromIPFS(hash string) (*OrganizationMetaData, error) {
	jsondata, err := ipfsutils.ReadFile(hash)
	if err != nil {
		return nil, err
	}
	return InitOrganizationMetaDataFromJson(jsondata)
}

func getMetaDataURI() []byte {
	// Blockchain call to get the hash of the metadata for the given Organization
	reg := getRegistryCaller()
	orgId := utils.StringToBytes32(config.GetString(config.OrganizationId))

	organizationRegistered, err := reg.GetOrganizationById(nil, orgId)
	if err != nil || !organizationRegistered.Found {
		zap.L().Panic("Error Retrieving contract details for the Given Organization, recheck blockchain provider endpoint", zap.String("OrganizationId", config.GetString(config.OrganizationId)), zap.Error(err))
	}
	return organizationRegistered.OrgMetadataURI[:]
}

// GetGroupIdString Get the Group ID the Daemon needs to associate itself to, requests belonging to a different group if will be rejected
func (metaData *OrganizationMetaData) GetGroupIdString() string {
	return metaData.daemonGroup.GroupID
}

// GetGroupId Return the group id in bytes
func (metaData *OrganizationMetaData) GetGroupId() [32]byte {
	return metaData.daemonGroupID
}

func (metaData *OrganizationMetaData) GetLicenseEndPoints() []string {
	return metaData.daemonGroup.LicenseEndpoints
}

// GetPaymentAddress retrieve the details of the payment address/ recipient address for current group.
func (metaData *OrganizationMetaData) GetPaymentAddress() common.Address {
	return metaData.recipientPaymentAddress
}

// GetPaymentExpirationThreshold get expiration threshold
func (metaData *OrganizationMetaData) GetPaymentExpirationThreshold() *big.Int {
	return metaData.daemonGroup.PaymentDetails.PaymentExpirationThreshold
}

// GetPaymentStorageEndPoints Get the End points of the Payment Storage used to update the storage state
func (metaData *OrganizationMetaData) GetPaymentStorageEndPoints() []string {
	return metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.Endpoints
}

// GetConnectionTimeOut Get the connection time out defined
func (metaData *OrganizationMetaData) GetConnectionTimeOut() (connectionTimeOut time.Duration) {
	connectionTimeOut, err := time.ParseDuration(metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.ConnectionTimeout)
	if err != nil {
		zap.L().Error(err.Error())
	}
	return connectionTimeOut
}

// GetRequestTimeOut Get the Request time out defined
func (metaData *OrganizationMetaData) GetRequestTimeOut() time.Duration {
	timeOut, err := time.ParseDuration(metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.RequestTimeout)
	if err != nil {
		zap.L().Error(err.Error())
	}
	return timeOut
}
