package config

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNetworkId(t *testing.T) {
	isolatedConfig(t)
	Vip().Set(BlockChainNetworkSelected, "sepolia")
	err := determineNetworkSelected([]byte(defaultBlockChainNetworkConfig))
	assert.Equal(t, err, nil)
	assert.Equal(t, GetNetworkId(), "11155111")
}

var defaultBlockChainNetworkConfig = `
{
  "local": {
    "ethereum_json_rpc_http_endpoint": "http://localhost:8545",
	"ethereum_json_rpc_ws_endpoint": "ws://localhost:443",
	"network_id": "42",
    "registry_address_key": "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2"
  },
  "main": {
    "ethereum_json_rpc_http_endpoint": "https://mainnet.infura.io/v3",
	"ethereum_json_rpc_ws_endpoint": "wss://mainnet.infura.io/v3",
    "network_id": "1"
  },
  "goerli": {
    "ethereum_json_rpc_http_endpoint": "https://goerli.infura.io/v3",
	"ethereum_json_rpc_ws_endpoint": "wss://goerli.infura.io/v3",
    "network_id": "5"
  },
  "sepolia": {
    "ethereum_json_rpc_http_endpoint": "https://sepolia.infura.io/v3/09027f4a13e841d48dbfefc67e7685d5",
	"ethereum_json_rpc_ws_endpoint": "wss://sepolia.infura.io/ws/v3/09027f4a13e841d48dbfefc67e7685d5",
    "network_id": "11155111"
  }
}`

func TestGetBlockChainEndPoint(t *testing.T) {
	isolatedConfig(t)
	Vip().Set(BlockChainNetworkSelected, "sepolia")
	err := determineNetworkSelected([]byte(defaultBlockChainNetworkConfig))
	assert.Equal(t, err, nil)
	assert.Contains(t, GetBlockChainHTTPEndPoint(), GetString(BlockChainNetworkSelected))
	assert.Contains(t, GetBlockChainWSEndPoint(), GetString(BlockChainNetworkSelected))
	assert.NotNil(t, GetNetworkId())
}

func TestGetRegistryAddress(t *testing.T) {
	isolatedConfig(t)

	err := determineNetworkSelected([]byte(defaultBlockChainNetworkConfig))
	assert.Nil(t, err)
	if GetString(BlockChainNetworkSelected) == "local" {
		assert.Equal(t, GetRegistryAddress(), "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2")
	} else {
		assert.NotNil(t, GetRegistryAddress())
	}
}

func TestReadFromFile(t *testing.T) {
	isolatedConfig(t)
	type args struct {
		filename string
	}
	tests := []struct {
		name string

		want    []byte
		wantErr bool
	}{
		{"s.txt", nil, true},
		{BlockChainNetworkFileName, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ReadFromFile(tt.name); err != nil {
				assert.Equal(t, true, tt.wantErr)
			}

		})
	}
}

func Test_SetBlockChainNetworkDetails(t *testing.T) {
	isolatedConfig(t)
	setBlockChainNetworkDetails(BlockChainNetworkFileName)
	assert.NotNil(t, GetRegistryAddress())
}

func Test_GetDetailsFromJsonOrConfig(t *testing.T) {
	isolatedConfig(t)

	dynamicBinding := map[string]any{}
	data := []byte(defaultBlockChainNetworkConfig)
	err := json.Unmarshal(data, &dynamicBinding)
	assert.Nil(t, err)
	var wantEthEndpoint = "https://sepolia.infura.io/v3/09027f4a13e841d48dbfefc67e7685d5"

	tests := []struct {
		name    string
		want    string
		network string
	}{
		{EthereumJsonRpcHTTPEndpointKey, wantEthEndpoint, "sepolia"},
		{RegistryAddressKey, "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2", "local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDetailsFromJsonOrConfig(dynamicBinding[tt.network].(map[string]any)[tt.name], tt.name); got != tt.want {
				t.Errorf("Failed getDetailsFromJsonOrConfig wanted %s but got %s", wantEthEndpoint, got)
			}
		})
	}
	assert.Equal(t, getDetailsFromJsonOrConfig(nil, OrganizationId), GetString(OrganizationId))
	assert.Equal(t, getDetailsFromJsonOrConfig(nil, ""), GetString(""))
}

func Test_setRegistryAddress(t *testing.T) {
	isolatedConfig(t)
	tests := []struct {
		networkID string
		wantErr   bool
	}{
		{"11155111", false},
		{"1", false},
		{"11155111_", true},
	}

	for _, tt := range tests {
		networkSelected.NetworkId = tt.networkID
		networkSelected.RegistryAddressKey = "" // reset for every test
		t.Run(networkSelected.NetworkId, func(t *testing.T) {
			if err := setRegistryAddress(); (err != nil) != tt.wantErr {
				t.Errorf("setRegistryAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_setBlockChainNetworkDetails(t *testing.T) {
	isolatedConfig(t)

	tests := []struct {
		name                 string
		wantErr              bool
		networkIdNameMapping string
	}{
		{BlockChainNetworkFileName, false, ""},
		{"", true, ""},
		{BlockChainNetworkFileName, true, "junk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			networkIdNameMapping = tt.networkIdNameMapping
			if err := setBlockChainNetworkDetails(tt.name); (err != nil) != tt.wantErr {
				t.Errorf("setBlockChainNetworkDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNetworkConfigurationOverrides(t *testing.T) {
	v := isolatedConfig(t)
	v.Set(EthereumJsonRpcHTTPEndpointKey, "https://rpc.example.test")
	v.Set(EthereumJsonRpcWSEndpointKey, "wss://rpc.example.test")
	const registry = "0x06A1D29e9FfA2415434A7A571235744F8DA2a514"
	v.Set(RegistryAddressKey, registry)
	require.NoError(t, determineNetworkSelected([]byte(defaultBlockChainNetworkConfig)))
	require.Equal(t, "11155111", GetNetworkId())
	require.Equal(t, "https://rpc.example.test", GetBlockChainHTTPEndPoint())
	require.Equal(t, "wss://rpc.example.test", GetBlockChainWSEndPoint())
	require.True(t, common.IsHexAddress(GetTokenAddress()))
	require.NoError(t, setRegistryAddress())
	require.Equal(t, registry, GetRegistryAddress(), "an explicit registry must be preserved")
}

func TestInvalidNetworkJSON(t *testing.T) {
	isolatedConfig(t)
	networkSelected.RegistryAddressKey = "unchanged"
	require.Error(t, determineNetworkSelected([]byte("{broken")))
	require.Equal(t, "unchanged", GetRegistryAddress())
	require.ErrorContains(t, deriveDataFromJSON([]byte("{broken")), "cannot parse the registry JSON")
	require.Equal(t, "unchanged", GetRegistryAddress())
}
