//go:generate protoc -I . ./configuration_service.proto --go-grpc_out=. --go_out=.
package configuration_service

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/authutils"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"math/big"
	"sort"
	"strings"
)

type ConfigurationService struct {
	//Has the authentication authenticationAddressList that will be used to validate any incoming requests for Configuration Service
	authenticationAddressList []common.Address
	broadcast                 *MessageBroadcaster
}

func (service ConfigurationService) mustEmbedUnimplementedConfigurationServiceServer() {
	//TODO implement me
	panic("implement me")
}

const (
	START_PROCESSING_ANY_REQUEST = 1
	STOP_PROCESING_ANY_REQUEST   = 0
)

// Set the list of allowed users
func getAuthenticationAddress() []common.Address {
	users := config.Vip().GetStringSlice(config.AuthenticationAddresses)
	userAddress := make([]common.Address, 0)
	if users == nil || len(users) == 0 {
		return userAddress
	}
	for _, user := range users {
		if !common.IsHexAddress(user) {
			fmt.Errorf("%v is not a valid hex address", user)

		} else {
			userAddress = append(userAddress, common.Address(common.BytesToAddress(common.FromHex(user))))
		}
	}
	return userAddress
}

// TO DO Separate PRs will be submitted to implement all the function below
func (service ConfigurationService) GetConfiguration(ctx context.Context, request *EmptyRequest) (response *ConfigurationResponse, err error) {
	//Authentication checks
	if err = service.authenticate("_GetConfiguration", request.Auth); err != nil {
		return nil, err
	}
	schema, err := service.buildSchemaDetails()
	if err != nil {
		return nil, err
	}
	response = &ConfigurationResponse{}
	response.Schema = schema
	response.CurrentConfiguration = getCurrentConfig()
	return response, nil
}

func (service ConfigurationService) UpdateConfiguration(ctx context.Context, request *UpdateRequest) (response *ConfigurationResponse, err error) {

	//Authentication checks
	if err = service.authenticate("_UpdateConfiguration", request.Auth); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("work in progress")
}

func (service ConfigurationService) StopProcessingRequests(ctx context.Context, request *EmptyRequest) (response *StatusResponse, err error) {
	//Authentication checks
	if err = service.authenticate("_StopProcessingRequests", request.Auth); err != nil {
		return nil, err
	}
	service.broadcast.trigger <- STOP_PROCESING_ANY_REQUEST
	return &StatusResponse{CurrentProcessingStatus: StatusResponse_HAS_STOPPED_PROCESSING_REQUESTS}, nil
}

func (service ConfigurationService) StartProcessingRequests(ctx context.Context, request *EmptyRequest) (response *StatusResponse, err error) {
	//Authentication checks
	if err = service.authenticate("_StartProcessingRequests", request.Auth); err != nil {
		return nil, err
	}
	service.broadcast.trigger <- START_PROCESSING_ANY_REQUEST
	return &StatusResponse{CurrentProcessingStatus: StatusResponse_REQUEST_IN_PROGRESS}, nil
}

func (service ConfigurationService) IsDaemonProcessingRequests(ctx context.Context, request *EmptyRequest) (response *StatusResponse, err error) {
	//Authentication checks
	if err = service.authenticate("_IsDaemonProcessingRequests", request.Auth); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("work in progress")
}

func (service ConfigurationService) authenticate(prefix string, auth *CallerAuthentication) (err error) {

	//Check if the Signature is not Expired only when block chain is enabled, current block number has no
	//meaning when block chain is in Disabled mode
	if config.GetBool(config.BlockchainEnabledKey) {
		if err = authutils.CompareWithLatestBlockNumber(big.NewInt(int64(auth.CurrentBlock))); err != nil {
			return err
		}
	}

	signerFromMessage, err := authutils.GetSignerAddressFromMessage(service.getMessageBytes(prefix, auth.CurrentBlock), auth.GetSignature())
	if err != nil {
		log.Error(err)
		return err
	}
	//Check if the Signature is Valid and Signed accordingly
	if err = service.checkAuthenticationAddress(*signerFromMessage); err != nil {
		return err
	}

	return nil
}

func (service ConfigurationService) checkAuthenticationAddress(signer common.Address) error {

	for _, user := range service.authenticationAddressList {
		if user == signer {
			return nil
		}
	}
	return fmt.Errorf("unauthorized access, %v is not authorized", signer.Hex())

}

// You will be able to start the Daemon without an Authentication Address for now
// but without Authentication authenticationAddressList , you cannot use the operator UI functionality
func NewConfigurationService(messageBroadcaster *MessageBroadcaster) *ConfigurationService {
	service := &ConfigurationService{
		authenticationAddressList: getAuthenticationAddress(),
		broadcast:                 messageBroadcaster,
	}
	return service
}

// Message format has been agreed to be as the below ( prefix,block number,and authenticating authenticationAddressList)
func (service *ConfigurationService) getMessageBytes(prefixMessage string, blockNumber uint64) []byte {
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		math.U256Bytes(big.NewInt(int64(blockNumber))),
	}, nil)
	return message
}

func (service *ConfigurationService) buildSchemaDetails() (schema *ConfigurationSchema, err error) {
	schema = &ConfigurationSchema{}

	detailsFromConfig, err := config.GetConfigurationSchema()
	schema.Details = make([]*ConfigurationParameter, 0)
	if err != nil {
		return nil, err
	}
	for _, eachConfig := range detailsFromConfig {
		schema.Details = append(schema.Details, convertToConfigurationParameter(eachConfig))
	}
	return schema, nil
}

func convertToConfigurationParameter(configSchema config.ConfigurationDetails) *ConfigurationParameter {
	configParam := &ConfigurationParameter{
		Name:          configSchema.Name,
		DefaultValue:  configSchema.DefaultValue,
		RestartDaemon: convertToUpdateAction(configSchema.RestartDaemon),
		Section:       configSchema.Section,
		Description:   configSchema.Description,
		Type:          convertToConfigurationType(configSchema.Type),
		Mandatory:     configSchema.Mandatory,
		Editable:      configSchema.Editable,
	}
	return configParam
}

func convertToConfigurationType(value string) ConfigurationParameter_Type {

	switch value {
	case "string":
		return ConfigurationParameter_STRING

	case "int":
		return ConfigurationParameter_INTEGER

	case "url":
		return ConfigurationParameter_URL

	case "bool":
		return ConfigurationParameter_BOOLEAN

	case "authenticationAddressList":
		return ConfigurationParameter_ADDRESS

	default:
		return ConfigurationParameter_STRING
	}

}

func convertToUpdateAction(value bool) ConfigurationParameter_UpdateAction {

	if value {
		return ConfigurationParameter_RESTART_REQUIRED
	}
	return ConfigurationParameter_NO_IMPACT
}

func getCurrentConfig() map[string]string {
	currentConfigMap := make(map[string]string, len(config.Vip().AllKeys()))
	keys := config.Vip().AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		if config.DisplayKeys[strings.ToUpper(key)] {
			currentConfigMap[key] = config.GetString(key)
		}

	}
	return currentConfigMap
}
