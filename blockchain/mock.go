package blockchain

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/mock"
)

type MockProcessor struct {
	mock.Mock
	EnabledFlag      bool
	multiPartyEscrow *MultiPartyEscrow
}

const MockedCurrentBlock = 100

func NewMockProcessor(enabled bool) *MockProcessor {
	return &MockProcessor{EnabledFlag: enabled}
}

func (m *MockProcessor) ReconnectToWsClient() error {
	return nil
}

func (m *MockProcessor) ConnectToWsClient() error {
	return nil
}

func (m *MockProcessor) Enabled() bool {
	return m.EnabledFlag
}

func (m *MockProcessor) EscrowContractAddress() common.Address {
	return common.Address{}
}

func (m *MockProcessor) MultiPartyEscrow() *MultiPartyEscrow {
	return &MultiPartyEscrow{}
}

func (m *MockProcessor) GetEthHttpClient() *ethclient.Client {
	return nil
}

func (m *MockProcessor) GetEthWSClient() *ethclient.Client {
	return nil
}

func (m *MockProcessor) CurrentBlock() (*big.Int, error) {
	return big.NewInt(MockedCurrentBlock), nil
}

func (m *MockProcessor) CompareWithLatestBlockNumber(blockNumberPassed *big.Int, allowedBlockChainDifference uint64) error {
	latestBlockNumber, err := m.CurrentBlock()
	if err != nil {
		return err
	}

	differenceInBlockNumber := blockNumberPassed.Sub(blockNumberPassed, latestBlockNumber)
	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > allowedBlockChainDifference {
		return fmt.Errorf("authentication failed as the signature passed has expired")
	}
	return nil
}

func (m *MockProcessor) HasIdentity() bool {
	return true
}

func (m *MockProcessor) Close() {
}

func (m *MockProcessor) MultiPartyEscrowChannel(channelID *big.Int) (channel *MultiPartyEscrowChannel, ok bool, err error) {

	//ch, err := processor.multiPartyEscrow.Channels(nil, channelID)
	//if err != nil {
	//	zap.L().Warn("Error while looking up for channel id in blockchain", zap.Error(err), channelIdField)
	//	return nil, false, err
	//}
	//if ch.Sender == zeroAddress {
	//	zap.L().Warn("Unable to find channel id in blockchain", channelIdField)
	//	return nil, false, nil
	//}

	channel = &MultiPartyEscrowChannel{
		Sender:     common.HexToAddress("0x000"),
		Recipient:  common.HexToAddress("0x000"),
		GroupId:    [32]byte{},
		Value:      big.NewInt(0),
		Nonce:      big.NewInt(0),
		Expiration: big.NewInt(0),
		Signer:     common.HexToAddress("0x000"),
	}

	return channel, true, nil
}
