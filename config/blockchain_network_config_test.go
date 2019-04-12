package config

import (
	"testing"

	"github.com/magiconair/properties/assert"
	assert2 "github.com/stretchr/testify/assert"
)

func TestGetNetworkId(t *testing.T) {

	//assert2.NotEqual(t,err,nil)
}

const DefaultBlockChainNetworkConfig string = `
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
	determineNetworkSelected([]byte(DefaultBlockChainNetworkConfig))
	assert.Matches(t, GetBlockChainEndPoint(), GetString(BlockChainNetworkSelected))
	assert2.NotEqual(t, GetNetworkId(), nil)
	assert2.NotEqual(t, GetRegistryAddress(), "")
}

func TestGetRegistryAddress(t *testing.T) {
	determineNetworkSelected([]byte(DefaultBlockChainNetworkConfig))
	if GetString(BlockChainNetworkSelected) == "local" {
		assert2.NotEqual(t, GetRegistryAddress(), "")
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
		{"BlockChainNetworkFileName", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadFromFile(tt.name)
			assert2.Nil(t, err)
			return

		})
	}
}

func Test_SetBlockChainNetworkDetails(t *testing.T) {
	setBlockChainNetworkDetails()
	assert2.NotNil(t,GetRegistryAddress())
}
