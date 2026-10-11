package config

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
	isolatedConfig(t)
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
	isolatedConfig(t)
	Vip().Set(AllowedUserFlag, true)
	Vip().Set(AllowedUserAddresses, []string{"0x39ee715b50e78a920120c1ded58b1a47f571ab75"})
	SetAllowedUsers()
	signer := common.BytesToAddress(common.FromHex("0x39ee715b50e78a920120c1ded58b1a47f571ab75"))

	assert.True(t, IsAllowedUser(&signer))
	signer = common.BytesToAddress(common.FromHex("0x49ee715b50e78a920120c1ded58b1a47f571ab75"))
	assert.False(t, IsAllowedUser(&signer))
}

func Test_validateMeteringChecks(t *testing.T) {
	isolatedConfig(t)
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

func TestConfigurationFileRoundTrip(t *testing.T) {
	v := isolatedConfig(t)
	source := filepath.Join(t.TempDir(), "source.json")
	require.NoError(t, os.WriteFile(source, []byte(`{
		"service_id":"example-service", "ipfs_timeout":42, "service_timeout":"2m",
		"blockchain_enabled":false, "trusted_free_call_signers":["signer"],
		"free_calls_per_address":{"user":5}
	}`), 0600))
	require.NoError(t, LoadConfig(source))
	require.Equal(t, "example-service", GetString(ServiceId))
	require.Equal(t, 42, GetInt(IpfsTimeout))
	require.Equal(t, big.NewInt(42), GetBigInt(IpfsTimeout))
	require.Equal(t, 2*time.Minute, GetDuration(ServiceTimeout))
	require.False(t, GetBool(BlockchainEnabledKey))
	require.Equal(t, []string{"signer"}, GetStringSlice(TrustedFreeCallSigners))
	require.Equal(t, "example-service", Get(ServiceId))
	require.Equal(t, map[string]any{"user": float64(5)}, GetStringMap(FreeCallsPerAddress))
	v.Set(ServiceId, "updated-service")
	destination := filepath.Join(t.TempDir(), "written.json")
	require.NoError(t, WriteConfig(destination))
	isolatedConfig(t)
	require.NoError(t, LoadConfig(destination))
	require.Equal(t, "updated-service", GetString(ServiceId))
	require.Equal(t, 42, GetInt(IpfsTimeout))
	require.Equal(t, 2*time.Minute, GetDuration(ServiceTimeout))
}

func TestConfigurationFileErrors(t *testing.T) {
	isolatedConfig(t)
	dir := t.TempDir()
	require.ErrorIs(t, LoadConfig(filepath.Join(dir, "missing.json")), os.ErrNotExist)
	malformed := filepath.Join(dir, "malformed.json")
	require.NoError(t, os.WriteFile(malformed, []byte("{broken"), 0600))
	require.Error(t, LoadConfig(malformed))
	require.Error(t, WriteConfig(filepath.Join(dir, "missing-directory", "config.json")))
}

func TestFreeCallsAllowed(t *testing.T) {
	const address = "0x06A1D29e9FfA2415434A7A571235744F8DA2a514"
	for _, tc := range []struct {
		name  string
		value any
		want  int
	}{
		{"integer", 7, 7}, {"JSON number", float64(9), 9},
		{"zero", 0, 0}, {"unlimited", "unlimited", -1}, {"infinity", "infinity", -1},
		{"numeric string", "12", 0}, {"invalid type", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := isolatedConfig(t)
			v.Set(FreeCallsPerAddress, map[string]any{address: tc.value})
			require.Equal(t, tc.want, GetFreeCallsAllowed(strings.ToLower(address)))
			require.Equal(t, tc.want, GetFreeCallsAllowed(strings.ToUpper(address)))
			require.Zero(t, GetFreeCallsAllowed("unknown"))
		})
	}
}

func TestTrustedFreeCallSigners(t *testing.T) {
	const first = "0x06A1D29e9FfA2415434A7A571235744F8DA2a514"
	const second = "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"
	for _, tc := range []struct {
		name  string
		value any
		want  []common.Address
	}{
		{"single address", first, []common.Address{common.HexToAddress(first)}},
		{"mixed list", []string{first, "invalid", second}, []common.Address{common.HexToAddress(first), common.HexToAddress(second)}},
		{"empty", []string{}, nil}, {"invalid", "invalid", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolatedConfig(t).Set(TrustedFreeCallSigners, tc.value)
			require.Equal(t, tc.want, GetTrustedFreeCallSignersAddresses())
		})
	}
}

func TestExperimentalConfiguration(t *testing.T) {
	v := isolatedConfig(t)
	require.Nil(t, GetExperimentalSettings())
	require.NoError(t, ReadConfigFromJsonString(v, `{"experimental":{
		"split_grpc_web":true,"use_original_cmux":true,"traffic_split":{"grpc":70,"http":30}
	}}`))
	settings := GetExperimentalSettings()
	require.NotNil(t, settings)
	require.True(t, settings.SplitWebgrpc)
	require.True(t, settings.UseOriginalCmux)
	require.NotNil(t, settings.TrafficSplit)
	require.Equal(t, uint32(70), settings.TrafficSplit.Grpc)
	require.Equal(t, uint32(30), settings.TrafficSplit.Http)
	require.NoError(t, ReadConfigFromJsonString(v, `{"experimental":{"split_grpc_web":false}}`))
	settings = GetExperimentalSettings()
	require.NotNil(t, settings)
	require.False(t, settings.SplitWebgrpc)
	require.Nil(t, settings.TrafficSplit)
	require.NoError(t, ReadConfigFromJsonString(v, `{"experimental":{"traffic_split":{"grpc":"invalid"}}}`))
	require.Nil(t, GetExperimentalSettings())
}

func TestBigIntConfiguration(t *testing.T) {
	v := isolatedConfig(t)
	for _, value := range []string{"0", "-42", "123456789012345678901234567890"} {
		v.Set("amount", value)
		got, err := GetBigIntFromViper(v, "amount")
		require.NoError(t, err)
		require.Equal(t, value, got.String())
	}
	v.Set("amount", "not a number")
	_, err := GetBigIntFromViper(v, "amount")
	require.Error(t, err)
}

func TestJSONConfigurationHelpers(t *testing.T) {
	v := NewJsonConfigFromString(`{"service_id":"test","ipfs_timeout":15}`)
	require.Equal(t, "test", v.GetString(ServiceId))
	require.Equal(t, 15, v.GetInt(IpfsTimeout))
	v = NewJsonConfigFromString("{broken")
	require.NotNil(t, v)
	require.Empty(t, v.AllSettings())
	data, err := ConvertStructToJSON(ConfigurationDetails{Name: "example", Mandatory: true})
	require.NoError(t, err)
	require.Contains(t, string(data), `"mandatory":true`)
	data, err = ConvertStructToJSON(make(chan int))
	require.Error(t, err)
	require.Nil(t, data)
}

func TestLogConfigFiltersKeysAndEmptyValues(t *testing.T) {
	isolatedConfig(t)
	// Use only these settings so the assertion also detects unexpected log entries.
	SetVip(NewJsonConfigFromString(`{"service_id":"visible","ssl_cert":"",
		"private_key_for_free_calls":"secret","ipfs_timeout":30,"blockchain_enabled":false}`))
	core, logs := observer.New(zap.InfoLevel)
	undo := zap.ReplaceGlobals(zap.New(core))
	t.Cleanup(undo)
	LogConfig()
	entries := logs.All()
	require.Len(t, entries, 3)
	require.Equal(t, "Final configuration: ", entries[0].Message)
	require.Equal(t, BlockchainEnabledKey, entries[1].Message)
	require.Equal(t, false, entries[1].ContextMap()["value"])
	require.Equal(t, ServiceId, entries[2].Message)
	require.Equal(t, "visible", entries[2].ContextMap()["value"])
}
