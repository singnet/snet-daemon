package blockchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestMockProcessor(t *testing.T) {
	mock := NewMockProcessor(true)

	assert.True(t, mock.Enabled())
	assert.NoError(t, mock.ReconnectToWsClient())
	assert.NoError(t, mock.ConnectToWsClient())
	assert.Equal(t, common.Address{}, mock.EscrowContractAddress())
	assert.NotNil(t, mock.MultiPartyEscrow())
	assert.Nil(t, mock.GetEthHttpClient())
	assert.Nil(t, mock.GetEthWSClient())
	assert.True(t, mock.HasIdentity())

	block, err := mock.CurrentBlock()
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(MockedCurrentBlock), block)

	channel, ok, err := mock.MultiPartyEscrowChannel(big.NewInt(1))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.NotNil(t, channel)
	assert.Equal(t, common.HexToAddress("0x000"), channel.Sender)
	assert.Equal(t, common.HexToAddress("0x000"), channel.Recipient)
	assert.Equal(t, big.NewInt(0), channel.Value)
	assert.Equal(t, big.NewInt(0), channel.Nonce)

	mock.Close()
}

func TestMockProcessorDisabled(t *testing.T) {
	mock := NewMockProcessor(false)

	assert.False(t, mock.Enabled())
}

func TestMockProcessorCompareWithLatestBlockNumber(t *testing.T) {
	mock := NewMockProcessor(true)

	// difference within the allowed limit
	assert.NoError(t, mock.CompareWithLatestBlockNumber(big.NewInt(MockedCurrentBlock+5), 10))

	// difference exceeds the allowed limit
	err := mock.CompareWithLatestBlockNumber(big.NewInt(MockedCurrentBlock+20), 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}
