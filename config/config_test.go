package config

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	INNER               = "inner"
	OUTER               = "outer"
	INNER_VALUE         = "inner-value"
	OUTER_INNER         = "outer.inner"
	INNER_DEFAULT       = "inner-default"
	INNER_DEFAULT_VALUE = "inner-default-value"
	SERVICE_ENDPOINT    = "http://127.0.0.1:8080"
	DAEMON_ENDPOINT     = "0.0.0.0:7000"
)

func TestCustomSubMap(t *testing.T) {
	var config = viper.New()
	config.Set(OUTER_INNER, INNER_VALUE)
	config.SetDefault("outer.inner-default", INNER_DEFAULT_VALUE)

	var sub = SubWithDefault(config, OUTER)

	assert.Equal(t, INNER_VALUE, sub.Get(INNER))
	assert.Equal(t, INNER_DEFAULT_VALUE, sub.Get(INNER_DEFAULT))
}

func TestCustomSubSingleValue(t *testing.T) {
	var config = viper.New()
	config.SetDefault("outer.inner-default", INNER_DEFAULT_VALUE)

	var sub = SubWithDefault(config, OUTER)

	assert.Equal(t, INNER_DEFAULT_VALUE, sub.Get(INNER_DEFAULT))
}

func TestCustomSubNoValue(t *testing.T) {
	var config = viper.New()
	config.SetDefault(OUTER, INNER_DEFAULT)

	var sub = SubWithDefault(config, OUTER)

	assert.NotNil(t, sub)
	assert.Equal(t, nil, sub.Get(INNER_DEFAULT))
}

func TestCustomSubNoKey(t *testing.T) {
	var config = viper.New()

	var sub = SubWithDefault(config, "unknown")

	assert.Nil(t, sub)
}

func TestCustomSubMapWithKeyInOtherCase(t *testing.T) {
	var config = viper.New()
	config.Set(OUTER_INNER, INNER_VALUE)
	config.SetDefault("OUTER.inner-DEFAULT", INNER_DEFAULT_VALUE)

	var sub = SubWithDefault(config, "OuTeR")

	assert.Equal(t, INNER_VALUE, sub.Get("iNnEr"))
	assert.Equal(t, INNER_DEFAULT_VALUE, sub.Get("iNnEr-DeFaUlT"))
}

const jsonConfigString = `
{
  "object": {
  	  "field": "value"
  },
  "array": [ "item-1", "item-2" ],
  "string-key": "string-value",
  "int-key": 42
}`

func assertConfigIsEqualToJsonConfigString(t *testing.T, config *viper.Viper) {
	assert.Equal(t, map[string]any{"field": "value"}, config.Get("object"))
	assert.Equal(t, "value", config.Get("object.field"))
	assert.Equal(t, []any{"item-1", "item-2"}, config.Get("array"))
	assert.Equal(t, "string-value", config.Get("string-key"))
	assert.Equal(t, 42, config.GetInt("int-key"))
}

func TestReadConfigFromJsonString(t *testing.T) {
	var config = viper.New()

	ReadConfigFromJsonString(config, jsonConfigString)

	assertConfigIsEqualToJsonConfigString(t, config)
}

func TestSetDefaultFromConfig(t *testing.T) {
	var config = viper.New()
	var defaults = viper.New()
	ReadConfigFromJsonString(defaults, jsonConfigString)

	SetDefaultFromConfig(config, defaults)

	assertConfigIsEqualToJsonConfigString(t, config)
}

func TestIsValidUrl(t *testing.T) {
	valid := IsValidUrl("")
	assert.Equal(t, valid, false)
	valid = IsValidUrl("http://test:8080")
	assert.Equal(t, valid, true)
}

func TestValidateEmail(t *testing.T) {
	valid := ValidateEmail("abc@gmail.com")
	assert.Equal(t, true, valid)
	valid = ValidateEmail("abc@xyz")
	assert.Equal(t, false, valid)
}

func TestValidateEndpoints(t *testing.T) {
	err := ValidateEndpoints("0.0.0.0:8080", SERVICE_ENDPOINT)
	assert.NotNil(t, err, "same endpoints not allowed")
	err = ValidateEndpoints("127.0.0.1:8080", SERVICE_ENDPOINT)
	assert.NotNil(t, err, "same endpoints not allowed")
	err = ValidateEndpoints("0.0.0.0:8080", "http://127.0.0.1:5000")
	assert.Nil(t, err)
	err = ValidateEndpoints("1.2.3.4:8080", SERVICE_ENDPOINT)
	assert.Nil(t, err)
	err = ValidateEndpoints("1.2.3.4:8080", "")
	assert.Equal(t, "service_endpoint is the endpoint of your AI service in the daemon config and needs to be a valid url", err.Error())
	err = ValidateEndpoints(DAEMON_ENDPOINT, "http://localhost:8080")
	assert.Nil(t, err)
	err = ValidateEndpoints(DAEMON_ENDPOINT, "http://localhost:8080")
	assert.Nil(t, err)
	err = ValidateEndpoints(DAEMON_ENDPOINT, "localhost:8080")
	assert.Nil(t, err)
	err = ValidateEndpoints(DAEMON_ENDPOINT, "http://somedomain")
	assert.Nil(t, err)
	err = ValidateEndpoints(DAEMON_ENDPOINT, "https://somedomain:8093")
	assert.Nil(t, err)
}

func TestAllowedUserChecks(t *testing.T) {
	err := allowedUserConfigurationChecks()
	assert.Equal(t, nil, err)
	vip.Set(AllowedUserFlag, true)
	err = allowedUserConfigurationChecks()
	assert.Equal(t, "a valid Address needs to be specified for the config allowed_user_addresses to ensure that, only these users can make calls", err.Error())
	vip.Set(AllowedUserAddresses, []string{"0x06A1D29e9FfA2415434A7A571235744F8DA2a514", "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"})
	err = allowedUserConfigurationChecks()
	assert.Equal(t, nil, err)
	vip.Set(AllowedUserAddresses, []string{"invalidHexaddress", "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"})
	err = allowedUserConfigurationChecks()
	assert.Equal(t, "invalidHexaddress is not a valid hex address", err.Error())
	vip.Set(BlockChainNetworkSelected, "main")
	err = allowedUserConfigurationChecks()
	assert.Equal(t, "service cannot be restricted to certain users when set up against Ethereum mainnet,the flag allowed_user_flag is set to true", err.Error())
}
func Test_IsAllowedUser(t *testing.T) {
	Vip().Set(AllowedUserFlag, true)
	Vip().Set(AllowedUserAddresses, []string{"0x39ee715b50e78a920120c1ded58b1a47f571ab75"})
	SetAllowedUsers()
	signer := common.BytesToAddress(common.FromHex("0x39ee715b50e78a920120c1ded58b1a47f571ab75"))

	assert.True(t, IsAllowedUser(&signer))
	signer = common.BytesToAddress(common.FromHex("0x49ee715b50e78a920120c1ded58b1a47f571ab75"))
	assert.False(t, IsAllowedUser(&signer))
}

func Test_validateMeteringChecks(t *testing.T) {
	vip.Set(MeteringEndpoint, "http://demo8325345.mockable.io")
	tests := []struct {
		name    string
		wantErr bool
		setup   func()
	}{
		{"", false, func() {}},
		{"", true, func() {
			vip.Set(MeteringEnabled, true)

			vip.Set(MeteringEndpoint, "badurl")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			if err := validateMeteringChecks(); (err != nil) != tt.wantErr {

				t.Errorf("validateMeteringChecks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
