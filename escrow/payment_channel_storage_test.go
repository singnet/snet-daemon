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
		replicaGroupID: func() (*big.Int, error) { return big.NewInt(123), nil },
		readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, nil
		},
	}
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
		replicaGroupID: func() (*big.Int, error) { return big.NewInt(123), nil },
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
		GroupId:    big.NewInt(123),
		Value:      big.NewInt(12345),
		Nonce:      big.NewInt(3),
		Expiration: big.NewInt(100),
	}
}

func (suite *BlockchainChannelReaderSuite) channel() *PaymentChannelData {
	return &PaymentChannelData{
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupID:          big.NewInt(123),
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
	reader.replicaGroupID = func() (*big.Int, error) { return big.NewInt(321), nil }

	channel, ok, err := reader.GetChannelStateFromBlockchain(suite.channelKey())

	assert.Equal(suite.T(), errors.New("Channel received belongs to another group of replicas, current group: 321, channel group: 123"), err)
	assert.False(suite.T(), ok)
	assert.Nil(suite.T(), channel)
}
