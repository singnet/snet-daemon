package escrow

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/singnet/snet-daemon/blockchain"
)

func NewBlockchainChannelReaderMock() *BlockchainChannelReader {
	return &BlockchainChannelReader{
		replicaGroupID: func() ([32]byte, error) { return [32]byte{123}, nil },
		readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, nil
		},
	}
}

type PaymentChannelStorageSuite struct {
	suite.Suite

	senderAddress    common.Address
	recipientAddress common.Address
	memoryStorage    *memoryStorage

	storage *PaymentChannelStorage
}

func (suite *PaymentChannelStorageSuite) SetupSuite() {
	suite.senderAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.memoryStorage = NewMemStorage()

	suite.storage = NewPaymentChannelStorage(suite.memoryStorage)
}

func (suite *PaymentChannelStorageSuite) SetupTest() {
	suite.memoryStorage.Clear()
}

func TestPaymentChannelStorageSuite(t *testing.T) {
	suite.Run(t, new(PaymentChannelStorageSuite))
}

func (suite *PaymentChannelStorageSuite) key(channelID int64) *PaymentChannelKey {
	return &PaymentChannelKey{ID: big.NewInt(channelID)}
}

func (suite *PaymentChannelStorageSuite) channel() *PaymentChannelData {
	return &PaymentChannelData{
		ChannelID:        big.NewInt(42),
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupID:          [32]byte{123},
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}
}

func (suite *PaymentChannelStorageSuite) TestGetAll() {
	channelA := suite.channel()
	suite.storage.Put(suite.key(41), channelA)
	channelB := suite.channel()
	suite.storage.Put(suite.key(42), channelB)

	channels, err := suite.storage.GetAll()

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.Equal(suite.T(), []*PaymentChannelData{channelA, channelB}, channels)
}

type BlockchainChannelReaderSuite struct {
	suite.Suite

	senderAddress    common.Address
	recipientAddress common.Address

	reader BlockchainChannelReader
}

func (suite *BlockchainChannelReaderSuite) SetupSuite() {
	suite.senderAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)

	suite.reader = BlockchainChannelReader{
		replicaGroupID: func() ([32]byte, error) { return [32]byte{123}, nil },
		readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return suite.mpeChannel(), true, nil
		},
	}
}

func TestBlockchainChannelReaderSuite(t *testing.T) {
	suite.Run(t, new(BlockchainChannelReaderSuite))
}

func (suite *BlockchainChannelReaderSuite) mpeChannel() *blockchain.MultiPartyEscrowChannel {
	return &blockchain.MultiPartyEscrowChannel{
		Sender:     suite.senderAddress,
		Recipient:  suite.recipientAddress,
		GroupId:    [32]byte{123},
		Value:      big.NewInt(12345),
		Nonce:      big.NewInt(3),
		Expiration: big.NewInt(100),
	}
}

func (suite *BlockchainChannelReaderSuite) channel() *PaymentChannelData {
	return &PaymentChannelData{
		ChannelID:        big.NewInt(42),
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupID:          [32]byte{123},
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}
}

func (suite *BlockchainChannelReaderSuite) channelKey() *PaymentChannelKey {
	return &PaymentChannelKey{
		ID: big.NewInt(42),
	}
}

func (suite *BlockchainChannelReaderSuite) TestGetChannelState() {
	channel, ok, err := suite.reader.GetChannelStateFromBlockchain(suite.channelKey())

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), suite.channel(), channel)
}

func (suite *BlockchainChannelReaderSuite) TestGetChannelStateIncorrectGroupId() {
	reader := suite.reader
	reader.replicaGroupID = func() ([32]byte, error) { return [32]byte{32}, nil }

	channel, ok, err := reader.GetChannelStateFromBlockchain(suite.channelKey())

	assert.Equal(suite.T(), errors.New("Channel received belongs to another group of replicas, current group: [32 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0], channel group: [123 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]"), err)
	assert.False(suite.T(), ok)
	assert.Nil(suite.T(), channel)
}
