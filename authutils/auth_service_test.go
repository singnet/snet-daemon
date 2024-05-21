// authutils package provides functions for all authentication and singature validation related operations
package authutils

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
)

func TestCompareWithLatestBlockNumber(t *testing.T) {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	currentBlockNum, _ := CurrentBlock()
	err := CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(13)))
	assert.Equal(t, err.Error(), "authentication failed as the signature passed has expired")

	currentBlockNum, _ = CurrentBlock()
	err = CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(1)))
	assert.Equal(t, nil, err)

}

func TestCheckAllowedBlockDifferenceForToken(t *testing.T) {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	currentBlockNum, _ := CurrentBlock()
	err := CheckIfTokenHasExpired(currentBlockNum.Sub(currentBlockNum, big.NewInt(20000)))
	assert.Equal(t, err.Error(), "authentication failed as the Free Call Token passed has expired")

	currentBlockNum, _ = CurrentBlock()
	err = CheckIfTokenHasExpired(currentBlockNum.Add(currentBlockNum, big.NewInt(20)))
	assert.Equal(t, nil, err)

}
