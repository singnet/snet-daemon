package configuration_service

import (
	"bytes"
	"math/big"

	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/singnet/snet-daemon/authutils"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration_Service_checkAuthenticationAddress(t *testing.T) {
	service := ConfigurationService{}
	err := service.checkAuthenticationAddress("")
	assert.Equal(t, err.Error(), "invalid hex address specified/missing for configuration 'authentication_address' ,this is a mandatory configuration required to be set up manually for remote updates")

	service.address = "0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e"

	//Now set the authentication_address in Daemon
	config.Vip().Set(config.AuthenticationAddress, "0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e")

	//Pass an invalid hex address
	err = service.checkAuthenticationAddress("0x5f41abf4c6a4633B1574f05e74b802cBF42a96e")
	assert.Equal(t, "0x5f41abf4c6a4633B1574f05e74b802cBF42a96e is an invalid hex Address", err.Error())

	//Pass a valid address , but not the one set up for authentication
	err = service.checkAuthenticationAddress("0xD6C6344f1D122dC6f4C1782A4622B683b9008081")
	assert.Equal(t, "unauthorized access, 0xD6C6344f1D122dC6f4C1782A4622B683b9008081 is not authorized", err.Error())

	//Pass the correct address to Authenticate
	err = service.checkAuthenticationAddress("0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e")
	assert.Nil(t, err)

}

func TestNewConfigurationService(t *testing.T) {
	config.Vip().Set(config.AuthenticationAddress, "sdsdds")
	service := NewConfigurationService(nil)
	assert.Equal(t, service.address, "")

	config.Vip().Set(config.AuthenticationAddress, "0xD6C6344f1D122dC6f4C1782A4622B683b9008081")
	service = NewConfigurationService(nil)
	assert.Equal(t, service.address, "0xD6C6344f1D122dC6f4C1782A4622B683b9008081")
}

func TestConfigurationService_authenticate(t *testing.T) {
	config.Vip().Set(config.BlockChainNetworkSelected, "ropsten")
	config.Validate()
	currBlk, _ := authutils.CurrentBlock()
	tests := []struct {
		address string
		auth    *CallerAuthentication
		wantErr bool
		prefix  string
		message string
	}{
		{address: "0xD6C6344f1D122dC6f4C1782A4622B683b9008081",
			auth: &CallerAuthentication{CurrentBlock: 2, UserAddress: "0xD6C6344f1D122dC6f4C1782A4622B683b9008081",
				Signature: nil}, wantErr: true, message: "authentication failed as the signature passed has expired", prefix: ""},
		{address: "0xD6C6344f1D122dC6f4C1782A4622B683b9008081",
			auth: &CallerAuthentication{CurrentBlock: currBlk.Uint64(), UserAddress: "0xF6C6344f1D122dC6f4C1782A4622B683b9008081",
				Signature: nil}, wantErr: true, message: "unauthorized access, 0xF6C6344f1D122dC6f4C1782A4622B683b9008081 is not authorized", prefix: ""},
		{address: "0xD6C6344f1D122dC6f4C1782A4622B683b9008081",
			auth: &CallerAuthentication{CurrentBlock: currBlk.Uint64(), UserAddress: "0xD6C6344f1D122dC6f4C1782A4622B683b9008081",
				Signature: nil}, wantErr: true, message: "incorrect signature length", prefix: ""},
	}
	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			service := ConfigurationService{
				address: tt.address,
			}
			err := service.authenticate(tt.prefix, tt.auth)
			if tt.wantErr {
				assert.Equal(t, tt.message, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestConfigurationService_getMessageBytes(t *testing.T) {
	service := ConfigurationService{
		address: "0xD6C6344f1D122dC6f4C1782A4622B683b9008081",
	}
	msg := service.getMessageBytes("ABC", 123)
	assert.Equal(t, bytes.Join([][]byte{
		[]byte("ABC"),
		abi.U256(big.NewInt(int64(123))),
		blockchain.HexToAddress("0xD6C6344f1D122dC6f4C1782A4622B683b9008081").Bytes(),
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
	assert.Equal(t, ConfigurationParameter_ADDRESS, convertToConfigurationType("address"))
	assert.Equal(t, ConfigurationParameter_INTEGER, convertToConfigurationType("int"))
	assert.Equal(t, ConfigurationParameter_URL, convertToConfigurationType("url"))
	assert.Equal(t, ConfigurationParameter_STRING, convertToConfigurationType("string"))
	assert.Equal(t, ConfigurationParameter_STRING, convertToConfigurationType("random"))

}

func Test_getCurrentConfig(t *testing.T) {

	currentConfig := getCurrentConfig()
	assert.NotEmpty(t, currentConfig[config.DaemonEndPoint])
}

func TestConfigurationService_buildSchemaDetails(t *testing.T) {
	service:= &ConfigurationService{}
	gotSchema, err := service.buildSchemaDetails()
	assert.Nil(t,err)
	assert.True(t, len(gotSchema.Details) > 2)

}
