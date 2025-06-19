package escrow

import (
	"errors"
	"github.com/singnet/snet-daemon/v6/storage"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func NewBlockchainChannelReaderMock() *BlockchainChannelReader {
	return &BlockchainChannelReader{

		readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, nil
		},
	}
}

type PaymentChannelStorageSuite struct {
	suite.Suite

	senderAddress    common.Address
	signerAddress    common.Address
	recipientAddress common.Address
	memoryStorage    *storage.MemoryStorage

	storage *PaymentChannelStorage
}

func (suite *PaymentChannelStorageSuite) SetupSuite() {
	suite.senderAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.signerAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.memoryStorage = storage.NewMemStorage()

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
		Signer:           suite.signerAddress,
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

func (suite *PaymentChannelStorageSuite) TestGetChannel() {
	expectedChannel := suite.channel()
	suite.storage.Put(suite.key(42), expectedChannel)
	channel, ok, err := suite.storage.Get(suite.key(42))

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.Equal(suite.T(), true, ok)
	assert.Equal(suite.T(), expectedChannel, channel)
}

type BlockchainChannelReaderSuite struct {
	suite.Suite

	senderAddress    common.Address
	recipientAddress common.Address
	signerAddress    common.Address

	reader BlockchainChannelReader
}

func (suite *BlockchainChannelReaderSuite) SetupSuite() {
	suite.senderAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.signerAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)

	suite.reader = BlockchainChannelReader{

		readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return suite.mpeChannel(), true, nil
		},
		recipientPaymentAddress: func() common.Address {
			address := suite.recipientAddress
			return address
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
		Signer:     suite.signerAddress,
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
		Signer:           suite.signerAddress,
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

func (suite *BlockchainChannelReaderSuite) TestGetChannelStateIncorrectRecipientAddress() {
	reader := suite.reader

	reader.recipientPaymentAddress = func() common.Address { return crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey) }
	channel, ok, err := reader.GetChannelStateFromBlockchain(suite.channelKey())
	assert.Equal(suite.T(), errors.New("recipient Address from org metadata does not Match on what was retrieved from Channel"), err)
	assert.False(suite.T(), ok)
	assert.Nil(suite.T(), channel)
}

func (suite *PaymentChannelStorageSuite) TestNewPaymentChannelStorage() {
	mpeStorage := storage.NewPrefixedAtomicStorage(storage.NewPrefixedAtomicStorage(suite.memoryStorage, "path1"), "path2")
	err := mpeStorage.Put("key1", "value1")
	assert.Nil(suite.T(), err)
	value, _, _ := mpeStorage.Get("key1")
	assert.Equal(suite.T(), value, "value1")
	values, err := suite.memoryStorage.GetByKeyPrefix("path1")
	assert.Equal(suite.T(), len(values), 1)
	assert.Equal(suite.T(), values[0], "value1")
	assert.Nil(suite.T(), err)
}

func (suite *PaymentChannelStorageSuite) TestExecuteTransaction() {
	t := suite.T()

	channelId := big.NewInt(1)
	price := big.NewInt(2)
	storage := NewPrepaidStorage(storage.NewPrefixedAtomicStorage(suite.memoryStorage, "path1"))
	service := NewPrePaidService(storage, nil, func() (bytes [32]byte, e error) {
		return [32]byte{123}, nil
	})

	err := service.UpdateUsage(channelId, big.NewInt(10), PLANNED_AMOUNT)
	assert.Nil(t, err)
	value, ok, err := service.GetUsage(PrePaidDataKey{ChannelID: channelId, UsageType: PLANNED_AMOUNT})
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, value.Amount, big.NewInt(10))

	err = service.UpdateUsage(channelId, price, USED_AMOUNT)
	assert.Nil(t, err)
	value, ok, err = service.GetUsage(PrePaidDataKey{ChannelID: channelId, UsageType: USED_AMOUNT})
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, value.Amount, price)

	err = service.UpdateUsage(channelId, price, REFUND_AMOUNT)
	assert.Nil(t, err)
	value, ok, err = service.GetUsage(PrePaidDataKey{ChannelID: channelId, UsageType: REFUND_AMOUNT})
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, value.Amount, price)

}
