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

var defaultBlockChainNetworkConfig string = `
{
  "local":{
    "ethereum_json_rpc_endpoint":"http://localhost:8545",
    "network_id":"42",
    "registry_address_key":"0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2"

  },
  "kovan":{
    "ethereum_json_rpc_endpoint":"https://kovan.infura.io",
    "network_id":"42"
  },
  "ropsten":{
    "ethereum_json_rpc_endpoint":"https://ropsten.infura.io",
    "network_id":"3"

  },
  "rinkeby":{
    "ethereum_json_rpc_endpoint":"https://rinkeby.infura.io",
    "network_id":"7"
  },
  "main":{
    "ethereum_json_rpc_endpoint":"https://mainnet.infura.io",
    "network_id":"1"
  }
}`

func TestGetBlockChainEndPoint(t *testing.T) {
	//ns := &NetworkSelected{}
	determineNetworkSelected([]byte(defaultBlockChainNetworkConfig))
	assert.Matches(t, GetBlockChainEndPoint(), GetString(BlockChainNetworkSelected))
	assert2.NotEqual(t, GetNetworkId(), nil)

}

func TestGetRegistryAddress(t *testing.T) {
	//ns := &NetworkSelected{}
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

	dynamicBinding := map[string]interface{}{}
	data := []byte(defaultBlockChainNetworkConfig)
	json.Unmarshal(data, &dynamicBinding)

	tests := []struct {
		name    string
		want    string
		network string
	}{
		{EthereumJsonRpcEndpointKey, "https://ropsten.infura.io", "ropsten"},
		{RegistryAddressKey, "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2", "local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDetailsFromJsonOrConfig(dynamicBinding[tt.network].(map[string]interface{})[tt.name], tt.name); got != tt.want {
				t.Errorf("getDetailsFromJsonOrConfig() = %v, want %v", got, tt.want)
			}
		})
	}
	assert.Equal(t, getDetailsFromJsonOrConfig(nil, OrganizationId), GetString(OrganizationId))

	assert.Equal(t, getDetailsFromJsonOrConfig(nil, ""), GetString(""))
}

func Test_setRegistryAddress(t *testing.T) {
	tests := []struct {
		fileName string
		registryJson string
		wantErr      bool
	}{
		{"resources/blockchain/node_modules/singularitynet-platform-contracts/networks/Registry.json","", false},
		{"junsdsdsk", "sdsd",true},
		{"junsdsdsk", "",true},
	}

	for _, tt := range tests {
		networkSelected.RegistryAddressKey = ""
		registryAddressJson = tt.registryJson
		t.Run(tt.registryJson, func(t *testing.T) {
			if err := setRegistryAddress(tt.fileName); (err != nil) != tt.wantErr {
				t.Errorf("setRegistryAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func Test_setBlockChainNetworkDetails(t *testing.T) {

	tests := []struct {
		name    string

		wantErr bool
		networkIdNameMapping string
	}{
		{BlockChainNetworkFileName,false,""},
		{"",true,""},
		{BlockChainNetworkFileName,true,"junk"},
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
