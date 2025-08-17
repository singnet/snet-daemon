package configuration_service

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration_Service_checkAuthenticationAddress(t *testing.T) {

	config.Vip().Set(config.AuthenticationAddresses, []string{"0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e", "0x06A1D29e9FfA2415434A7A571235744F8DA2a514", "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"})
	service := ConfigurationService{authenticationAddressList: getAuthenticationAddress()}
	err := service.checkAuthenticationAddress(common.BytesToAddress(common.FromHex("0x39ee715b50e78a920120C1dED58b1a47F571AB75")))
	assert.Equal(t, err.Error(), "unauthorized access, 0x39ee715b50e78a920120C1dED58b1a47F571AB75 is not authorized")
	//Pass an invalid hex authenticationAddressList
	err = service.checkAuthenticationAddress(common.BytesToAddress(common.FromHex("0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e")))
	assert.Nil(t, err)

}

func TestNewConfigurationService(t *testing.T) {
	config.Vip().Set(config.AuthenticationAddresses, "")
	service := NewConfigurationService(nil, blockchain.NewMockProcessor(true))
	assert.Equal(t, 0, len(service.authenticationAddressList))
	config.Vip().Set(config.AuthenticationAddresses, []string{"0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e", "0x06A1D29e9FfA2415434A7A571235744F8DA2a514", "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"})
	service = NewConfigurationService(nil, blockchain.NewMockProcessor(true))
	assert.Equal(t, 3, len(service.authenticationAddressList))
}

func TestConfigurationService_authenticate(t *testing.T) {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	service := NewConfigurationService(nil, blockchain.NewMockProcessor(true))
	config.Validate()
	privateKey, _ := crypto.GenerateKey()
	currBlock, _ := service.blockchainProc.CurrentBlock()
	publicKey := crypto.PubkeyToAddress(privateKey.PublicKey)
	msg := service.getMessageBytes("__GetConfiguration", currBlock.Uint64())
	sig := utils.GetSignature(msg, privateKey)
	auth := &CallerAuthentication{CurrentBlock: currBlock.Uint64(), Signature: sig}
	err := service.authenticate("__GetConfiguration", auth)
	assert.Contains(t, err.Error(), "unauthorized access")
	config.Vip().Set(config.AuthenticationAddresses, []string{publicKey.Hex(), "0x06A1D29e9FfA2415434A7A571235744F8DA2a514", "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"})
	service = NewConfigurationService(nil, blockchain.NewMockProcessor(true))
	err = service.authenticate("__GetConfiguration", auth)
	assert.Nil(t, err)
}

func TestConfigurationService_getMessageBytes(t *testing.T) {
	config.Vip().Set(config.AuthenticationAddresses, []string{"0xD6C6344f1D122dC6f4C1782A4622B683b9008081", "0x06A1D29e9FfA2415434A7A571235744F8DA2a514", "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"})
	service := NewConfigurationService(nil, blockchain.NewMockProcessor(true))
	msg := service.getMessageBytes("ABC", 123)
	assert.Equal(t, bytes.Join([][]byte{
		[]byte("ABC"),
		math.U256Bytes(big.NewInt(int64(123))),
	}, nil), msg)
}

func Test_convertToConfigurationParameter(t *testing.T) {

	configParam := convertToConfigurationParameter(config.ConfigurationDetails{
		RestartDaemon: true,
		Type:          "url",
		Editable:      true,
		Description:   "Testing",
		Mandatory:     true,
		Section:       "Blockchain",
		DefaultValue:  "testvalue",
		Name:          "testname",
	})
	assert.Equal(t, "testname", configParam.Name)
	assert.Equal(t, ConfigurationParameter_RESTART_REQUIRED, configParam.RestartDaemon)
}

func Test_convertToUpdateAction(t *testing.T) {
	assert.Equal(t, ConfigurationParameter_RESTART_REQUIRED, convertToUpdateAction(true))
	assert.Equal(t, ConfigurationParameter_NO_IMPACT, convertToUpdateAction(false))

}

func Test_convertToConfigurationType(t *testing.T) {
	assert.Equal(t, ConfigurationParameter_BOOLEAN, convertToConfigurationType("bool"))
	assert.Equal(t, ConfigurationParameter_ADDRESS, convertToConfigurationType("authenticationAddressList"))
	assert.Equal(t, ConfigurationParameter_INTEGER, convertToConfigurationType("int"))
	assert.Equal(t, ConfigurationParameter_URL, convertToConfigurationType("url"))
	assert.Equal(t, ConfigurationParameter_STRING, convertToConfigurationType("string"))
	assert.Equal(t, ConfigurationParameter_STRING, convertToConfigurationType("random"))
}

func Test_getCurrentConfig(t *testing.T) {

	currentConfig := getCurrentConfig()
	assert.NotEmpty(t, currentConfig[config.DaemonEndpoint])
	config.Vip().Set(config.PvtKeyForMetering, "HIDDEN")
	currentConfig = getCurrentConfig()
	assert.Empty(t, currentConfig[config.PvtKeyForMetering])
}

func TestConfigurationService_buildSchemaDetails(t *testing.T) {
	service := &ConfigurationService{}
	gotSchema, err := service.buildSchemaDetails()
	assert.Nil(t, err)
	assert.True(t, len(gotSchema.Details) > 2)
}
