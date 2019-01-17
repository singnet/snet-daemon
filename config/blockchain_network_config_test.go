package config

import (
	"github.com/magiconair/properties/assert"
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestGetRegistryAddress(t *testing.T) {
	setAndValidateBlockChainNetworkDetails()
	assert2.NotEqual(t,GetRegistryAddress(),nil)
}


func TestGetBlockChainEndPoint(t *testing.T) {
	determineNetworkSelected([]byte(DefaultBlockChainNetworkConfig))
	assert.Matches(t,GetBlockChainEndPoint(),GetString(BlockChainNetworkSelected))
	assert2.NotEqual(t,GetNetworkId(),nil)
	assert2.NotEqual(t,GetRegistryAddress(),"")
}


func TestDetermineNetworkSelected(t *testing.T) {
	determineNetworkSelected([]byte(DefaultBlockChainNetworkConfig))
	if GetString(BlockChainNetworkSelected) == "local" {
		assert2.NotEqual(t, GetRegistryAddress(), "")
	} else {
		assert2.NotEqual(t, GetRegistryAddress(), nil)
	}
}

func TestReadFromFile(t *testing.T) {
	data,err:=ReadFromFile("../blockchain_network_config.json")
	assert2.NotEqual(t,data,nil)
	assert.Equal(t,err,nil)
}

func TestGetRegistryAddressFromJson (t *testing.T) {

	address := getRegistryAddressFromJson([]byte("{\"999999\":{\"events\":{},\"links\":{},\"address\":\"0xe331bf20044a5b24c1a744abc90c1fd711d2c08d\",\"transactionHash\":\"0xb98bcff3cfd65916b50f85f3bb89c83252b24b74f42ede4c4badd2456cf2bf9a\"}}"))
	assert.Equal(t,address,"0xe331bf20044a5b24c1a744abc90c1fd711d2c08d")
}

