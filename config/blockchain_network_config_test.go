package config

import (
	"github.com/magiconair/properties/assert"
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNetworkId(t *testing.T) {

	//assert2.NotEqual(t,err,nil)
}


func TestGetBlockChainEndPoint(t *testing.T) {
	determineNetworkSelected([]byte(DefaultBlockChainNetworkConfig))
	assert.Matches(t,GetBlockChainEndPoint(),GetString(BlockChainNetworkSelected))
	assert2.NotEqual(t,GetNetworkId(),nil)
	assert2.NotEqual(t,GetRegistryAddress(),"")
}


func TestGetRegistryAddress(t *testing.T) {
	determineNetworkSelected([]byte(DefaultBlockChainNetworkConfig))
	if GetString(BlockChainNetworkSelected) == "local" {
		assert2.NotEqual(t, GetRegistryAddress(), "")
	} else {
		assert2.NotEqual(t, GetRegistryAddress(), nil)
	}
}

