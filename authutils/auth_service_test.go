//  authutils package provides functions for all authentication and singature validation related operations
package authutils

import (
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestCompareWithLatestBlockNumber(t *testing.T) {
	config.Vip().Set(config.EthereumJsonRpcEndpointKey, "https://ropsten.infura.io")
	currentBlockNum, _ := CurrentBlock()
	err := CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(13)))
	assert.Equal(t, err.Error(), "difference between the latest block chain number and the block number passed is 13 ")

	currentBlockNum, _ = CurrentBlock()
	err = CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(1)))
	assert.Equal(t, nil, err)

}
