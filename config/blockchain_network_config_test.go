package config

import (
	"encoding/json"
	"testing"

	"github.com/magiconair/properties/assert"
	assert2 "github.com/stretchr/testify/assert"
)

func TestGetNetworkId(t *testing.T) {

	//assert2.NotEqual(t,err,nil)
}

var defaultBlockChainNetworkConfig = `
{
  "local": {
    "ethereum_json_rpc_endpoint": "http://localhost:8545",
    "network_id": "42",
    "registry_address_key": "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2"
  },
  "main": {
    "ethereum_json_rpc_endpoint": "https://mainnet.infura.io/v3",
    "network_id": "1"
  },
  "goerli": {
    "ethereum_json_rpc_endpoint": "https://goerli.infura.io/v3",
    "network_id": "5"
  },
  "sepolia": {
    "ethereum_json_rpc_endpoint": "https://sepolia.infura.io/v3",
    "network_id": "11155111"
  }
}`

func TestGetBlockChainEndPoint(t *testing.T) {
	Vip().Set(BlockChainNetworkSelected, "local")
	determineNetworkSelected([]byte(defaultBlockChainNetworkConfig))

	assert.Matches(t, GetBlockChainEndPoint(), GetString(BlockChainNetworkSelected))
	assert2.NotEqual(t, GetNetworkId(), nil)

}

func TestGetRegistryAddress(t *testing.T) {

	determineNetworkSelected([]byte(defaultBlockChainNetworkConfig))
	if GetString(BlockChainNetworkSelected) == "local" {
		assert2.Equal(t, GetRegistryAddress(), "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2")
	} else {
		assert2.NotEqual(t, GetRegistryAddress(), nil)
	}
}

func TestReadFromFile(t *testing.T) {
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
				assert2.Equal(t, true, tt.wantErr)
			}

		})
	}
}

func Test_SetBlockChainNetworkDetails(t *testing.T) {
	setBlockChainNetworkDetails(BlockChainNetworkFileName)
	assert2.NotNil(t, GetRegistryAddress())
}

func Test_GetDetailsFromJsonOrConfig(t *testing.T) {

	dynamicBinding := map[string]any{}
	data := []byte(defaultBlockChainNetworkConfig)
	json.Unmarshal(data, &dynamicBinding)

	tests := []struct {
		name    string
		want    string
		network string
	}{
		{EthereumJsonRpcEndpointKey, "https://sepolia.infura.io/v3", "sepolia"},
		{RegistryAddressKey, "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2", "local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDetailsFromJsonOrConfig(dynamicBinding[tt.network].(map[string]any)[tt.name], tt.name); got != tt.want {
				t.Errorf("getDetailsFromJsonOrConfig() = %v, want %v", got, tt.want)
			}
		})
	}
	assert.Equal(t, getDetailsFromJsonOrConfig(nil, OrganizationId), GetString(OrganizationId))

	assert.Equal(t, getDetailsFromJsonOrConfig(nil, ""), GetString(""))
}

func Test_setRegistryAddress(t *testing.T) {
	tests := []struct {
		networkID string
		wantErr   bool
	}{
		{"11155111", false},
		{"5", false},
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
