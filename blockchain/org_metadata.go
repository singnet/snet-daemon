package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"math/big"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/ipfsutils"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

// OrganizationMetaData is off-chain JSON referenced by an organization's Registry metadata URI.
// Its named replica groups provide group IDs, provider payment addresses, and
// storage settings shared by services using a group. daemon_group_name selects
// the local group; MPE recipient/group checks are performed separately by escrow.
type OrganizationMetaData struct {
	OrgName string  `json:"org_name"`
	OrgID   string  `json:"org_id"`
	Groups  []Group `json:"groups"`
	// Derived fields belong to the group selected by daemon_group_name.
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

// Group defines one organization-level replica group and its shared payment,
// storage, and license-server settings. These values are read from metadata JSON.
type Group struct {
	GroupName        string   `json:"group_name"`
	GroupID          string   `json:"group_id"`
	PaymentDetails   Payment  `json:"payment"`
	LicenseEndpoints []string `json:"license_server_endpoints"`
}

// PaymentChannelStorageClient contains etcd connection settings shared by the
// daemon's scoped payment, prepaid, free-call, and training storage.
type PaymentChannelStorageClient struct {
	ConnectionTimeout string   `json:"connection_timeout" mapstructure:"connection_timeout"`
	RequestTimeout    string   `json:"request_timeout" mapstructure:"request_timeout"`
	Endpoints         []string `json:"endpoints"`
}

// InitOrganizationMetaDataFromJson parses metadata, selects daemon_group_name,
// decodes its group ID, and validates the required recipient/storage settings.
func InitOrganizationMetaDataFromJson(jsonData []byte) (metaData *OrganizationMetaData, err error) {
	metaData = new(OrganizationMetaData)
	err = json.Unmarshal(jsonData, &metaData)
	if err != nil {
		zap.L().Error("Error in unmarshalling metadata json", zap.Error(err), zap.Any("jsondata", jsonData))
		return nil, err
	}

	// Check the required fields after selecting and deriving the local group.
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
		err = fmt.Errorf("Mandatory field : ETCD Client Endpoints are missing for the Group %v ", metaData.daemonGroup.GroupName)
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
	// Normalize the selected group's payment-address text and retain its byte value.
	metaData.daemonGroup.PaymentDetails.PaymentAddress = utils.ToChecksumAddressStr(metaData.daemonGroup.PaymentDetails.PaymentAddress)
	metaData.recipientPaymentAddress = common.HexToAddress(metaData.daemonGroup.PaymentDetails.PaymentAddress)

	return err
}

// getDaemonGroup selects the organization group by configured name, not by an
// MPE channel lookup or Registry ownership.
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

// GetOrganizationMetaData resolves the organization's Registry metadata URI and
// loads its JSON through IPFS. Blockchain-disabled mode returns an empty group.
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

// GetOrganizationMetaDataContext keeps hot-reload reads cancellable until all data is fetched.
func GetOrganizationMetaDataContext(ctx context.Context) (*OrganizationMetaData, error) {
	timeout := config.GetDuration(config.ServiceTimeout)
	if timeout <= 0 {
		timeout = 100 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, err := CreateHTTPEthereumClientContext(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	registry, err := NewRegistryCaller(getRegistryAddressKey(), client.EthClient)
	if err != nil {
		return nil, err
	}
	org, err := registry.GetOrganizationById(&bind.CallOpts{Context: ctx}, utils.StringToBytes32(config.GetString(config.OrganizationId)))
	if err != nil {
		return nil, err
	}
	if !org.Found {
		return nil, fmt.Errorf("organization not found")
	}
	data, err := ipfsutils.ReadFileContext(ctx, string(org.OrgMetadataURI[:]))
	if err != nil {
		return nil, err
	}
	return InitOrganizationMetaDataFromJson(data)
}

func GetOrganizationMetaDataFromIPFS(hash string) (*OrganizationMetaData, error) {
	jsondata, err := ipfsutils.ReadFile(hash)
	if err != nil {
		return nil, err
	}
	return InitOrganizationMetaDataFromJson(jsondata)
}

func getMetaDataURI() []byte {
	// Registry stores the metadata URI; the JSON itself is loaded off chain.
	reg := getRegistryCaller()
	orgId := utils.StringToBytes32(config.GetString(config.OrganizationId))

	organizationRegistered, err := reg.GetOrganizationById(nil, orgId)
	if err != nil || !organizationRegistered.Found {
		zap.L().Panic("Error Retrieving contract details for the Given Organization, recheck blockchain provider endpoint", zap.String("OrganizationId", config.GetString(config.OrganizationId)), zap.Error(err))
	}
	return organizationRegistered.OrgMetadataURI[:]
}

// GetGroupIdString returns the selected group's original metadata ID string.
// It does not validate a request; free-call payloads use this string as text.
func (metaData *OrganizationMetaData) GetGroupIdString() string {
	return metaData.daemonGroup.GroupID
}

// GetGroupId returns the decoded bytes32 group ID used in MPE comparisons and
// canonical storage keys. GetGroupIdString retains the metadata's textual form.
func (metaData *OrganizationMetaData) GetGroupId() [32]byte {
	return metaData.daemonGroupID
}

func (metaData *OrganizationMetaData) GetLicenseEndPoints() []string {
	return metaData.daemonGroup.LicenseEndpoints
}

// GetPaymentAddress returns the selected group's provider payment address,
// expected as an MPE channel recipient and provider-control request signer.
func (metaData *OrganizationMetaData) GetPaymentAddress() common.Address {
	return metaData.recipientPaymentAddress
}

// GetPaymentExpirationThreshold returns the safety margin in blocks used when
// accepting channel payments, not the request-signature freshness allowance.
func (metaData *OrganizationMetaData) GetPaymentExpirationThreshold() *big.Int {
	return metaData.daemonGroup.PaymentDetails.PaymentExpirationThreshold
}

// GetPaymentStorageEndPoints returns the selected group's shared etcd endpoints.
func (metaData *OrganizationMetaData) GetPaymentStorageEndPoints() []string {
	return metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.Endpoints
}

// GetConnectionTimeOut parses the group's connection timeout. An invalid duration
// is logged and produces a zero duration.
func (metaData *OrganizationMetaData) GetConnectionTimeOut() (connectionTimeOut time.Duration) {
	connectionTimeOut, err := time.ParseDuration(metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.ConnectionTimeout)
	if err != nil {
		zap.L().Error(err.Error())
	}
	return connectionTimeOut
}

// GetRequestTimeOut parses the group's request timeout. An invalid duration is
// logged and produces a zero duration.
func (metaData *OrganizationMetaData) GetRequestTimeOut() time.Duration {
	timeOut, err := time.ParseDuration(metaData.daemonGroup.PaymentDetails.PaymentChannelStorageClient.RequestTimeout)
	if err != nil {
		zap.L().Error(err.Error())
	}
	return timeOut
}
