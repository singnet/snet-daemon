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
	config.Validate()
	currentBlockNum, _ := CurrentBlock()
	err := CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(13)))
	assert.Equal(t, err.Error(), "authentication failed as the signature passed has expired")

	currentBlockNum, _ = CurrentBlock()
	err = CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(1)))
	assert.Equal(t, nil, err)

}
