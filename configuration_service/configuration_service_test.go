package configuration_service

import (
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration_Service_checkAuthenticationAddress(t *testing.T) {
	service := ConfigurationService{}
	err := service.checkAuthenticationAddress("")
	assert.Equal(t, err.Error(), "invalid hex address specified/missing for configuration 'authentication_address' ,this is a mandatory configuration required to be set up manually for remote updates.")

	service.address = "0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e"

	//Now set the authentication_address in Daemon
	config.Vip().Set(config.AuthenticationAddress, "0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e")

	//Pass an invalid hex address
	err = service.checkAuthenticationAddress("0x5f41abf4c6a4633B1574f05e74b802cBF42a96e")
	assert.Equal(t, "0x5f41abf4c6a4633B1574f05e74b802cBF42a96e is an invalid hex Address", err.Error())

	//Pass a valid address , but not the one set up for authentication
	err = service.checkAuthenticationAddress("0xD6C6344f1D122dC6f4C1782A4622B683b9008081")
	assert.Equal(t, "Unauthorized access, 0xD6C6344f1D122dC6f4C1782A4622B683b9008081 is not authorized", err.Error())

	//Pass the correct address to Authenticate
	err = service.checkAuthenticationAddress("0x4Af41abf4c6a4633B1574f05e74b802cBF42a96e")
	assert.Nil(t, err)

}

func TestNewConfigurationService(t *testing.T) {
	config.Vip().Set(config.AuthenticationAddress, "sdsdds")
	service := NewConfigurationService()
	assert.Equal(t,service.address,"")

	config.Vip().Set(config.AuthenticationAddress, "0xD6C6344f1D122dC6f4C1782A4622B683b9008081")
	service = NewConfigurationService()
	assert.Equal(t,service.address,"0xD6C6344f1D122dC6f4C1782A4622B683b9008081")
}
