// authutils package provides functions for all authentication and signature validation related operations
package authutils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestCompareWithLatestBlockNumber(t *testing.T) {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	currentBlockNum, err := CurrentBlock()
	assert.Nil(t, err)

	err = CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(2)))
	assert.Equal(t, nil, err)

	err = CompareWithLatestBlockNumber(currentBlockNum.Sub(currentBlockNum, big.NewInt(13)))
	assert.Equal(t, err.Error(), "authentication failed as the signature passed has expired")
}

func TestVerifyAddress(t *testing.T) {
	var addr = common.Address(common.FromHex("0x7DF35C98f41F3AF0DF1DC4C7F7D4C19A71DD079F"))
	var addrLowCase = common.Address(common.FromHex("0x7df35c98f41f3af0df1dc4c7f7d4c19a71Dd079f"))
	assert.Nil(t, VerifyAddress(addr, addrLowCase))
}
