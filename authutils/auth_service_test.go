//  authutils package provides functions for all authentication and singature validation related operations
package authutils

import (
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
)

func TestCompareWithLatestBlockNumber(t *testing.T) {
	config.Vip().Set(config.BlockChainNetworkSelected, "ropsten")
	config.Validate()
	currentBlockNum, _ := CurrentBlock()
	err := CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(13)))
	assert.Equal(t, err.Error(), "authentication failed as the signature passed has expired")

	currentBlockNum, _ = CurrentBlock()
	err = CompareWithLatestBlockNumber(currentBlockNum.Add(currentBlockNum, big.NewInt(1)))
	assert.Equal(t, nil, err)

}

func Test_getPrivateKeyForMetering(t *testing.T) {
  config.Vip().Set(config.PvtKeyForMetering,"063C00D18E147F4F734846E47FE6598FC7A6D56307862F7EDC92B9F43CC27EDD")
  key,err := getPrivateKeyForMetering()
  if err == nil {
	  assert.Equal(t, crypto.PubkeyToAddress(key.PublicKey).String(), "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
	  assert.NotNil(t, key)
	  assert.Nil(t, err)

	  bytesForMetering := signForMeteringValidation(key, big.NewInt(123), MeteringPrefix)
	  signature := getSignature(bytesForMetering, key)
	  signer, err := GetSignerAddressFromMessage(bytesForMetering, signature)
	  assert.Equal(t, signer.String(), "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
	  assert.Nil(t, err)
  }

}
