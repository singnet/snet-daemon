//go:generate protoc -I . ./configuration_service.proto --go_out=plugins=grpc:.
package configuration_service

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/authutils"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"math/big"
	"sort"
	"strings"
)

type ConfigurationService struct {
	//Has the authentication address that will be used to validate any incoming requests for Configuration Service
	address string
}

//TO DO Separate PRs will be submitted to implement all the function below
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
	return nil, fmt.Errorf("work in progress")
}

func (service ConfigurationService) StartProcessingRequests(ctx context.Context, request *EmptyRequest) (response *StatusResponse, err error) {
	//Authentication checks
	if err = service.authenticate("_StartProcessingRequests", request.Auth); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("work in progress")
}

func (service ConfigurationService) IsDaemonProcessingRequests(ctx context.Context, request *EmptyRequest) (response *StatusResponse, err error) {
	//Authentication checks
	if err = service.authenticate("_IsDaemonProcessingRequests", request.Auth); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("work in progress")
}

func (service ConfigurationService) authenticate(prefix string, auth *CallerAuthentication) (err error) {

	//Check if the address passed is the expected authentication address
	if err = service.checkAuthenticationAddress(auth.UserAddress); err != nil {
		return err
	}

	//Check if the Signature is not Expired
	if err = authutils.CompareWithLatestBlockNumber(big.NewInt(int64(auth.CurrentBlock))); err != nil {
		return err
	}

	//Check if the Signature is Valid and Signed accordingly
	if err = authutils.VerifySigner(service.getMessageBytes(prefix, auth.CurrentBlock),
		auth.GetSignature(), blockchain.HexToAddress(service.address)); err != nil {
		return err
	}

	return nil
}

func (service ConfigurationService) checkAuthenticationAddress(address string) error {

	if !common.IsHexAddress(service.address) {
		return fmt.Errorf("invalid hex address specified/missing for configuration 'authentication_address' ,this is a mandatory configuration required to be set up manually for remote updates")
	}
	if !common.IsHexAddress(address) {
		return fmt.Errorf("%v is an invalid hex Address", address)
	} else if strings.Compare(service.address, address) != 0 {
		return fmt.Errorf("unauthorized access, %v is not authorized", address)
	}
	return nil
}

//You will be able to start the Daemon without an Authentication Address for now
//but without Authentication address , you cannot use the operator UI functionality
func NewConfigurationService() *ConfigurationService {
	service := &ConfigurationService{
		address: config.GetString(config.AuthenticationAddress),
	}
	authAddress := config.GetString(config.AuthenticationAddress)
	//Make sure the address is a valid Hex Address
	if !common.IsHexAddress(authAddress) {
		service.address = ""
		log.Errorf("invalid hex address specified/missing for 'authentication_address' in configuration , you cannot make remote update to current configurations")
	}

	return service
}

//Message format has been agreed to be as the below ( prefix,block number,and authenticating address)
func (service *ConfigurationService) getMessageBytes(prefixMessage string, blocknumber uint64) []byte {
	message := bytes.Join([][]byte{
		[]byte (prefixMessage),
		abi.U256(big.NewInt(int64(blocknumber))),
		blockchain.HexToAddress(service.address).Bytes(),
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

	case "address":
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
		currentConfigMap[key] = config.GetString(key)
	}
	return currentConfigMap
}