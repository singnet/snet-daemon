package escrow

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type stateServiceTestType struct {
	service            PaymentChannelStateService
	senderAddress      common.Address
	signerPrivateKey   *ecdsa.PrivateKey
	signerAddress      common.Address
	channelServiceMock *paymentChannelServiceMock
	paymentStorage     *PaymentStorage

	defaultChannelId   *big.Int
	defaultChannelKey  *PaymentChannelKey
	defaultChannelData *PaymentChannelData
	defaultRequest     *ChannelStateRequest
	defaultReply       *ChannelStateReply
}

var stateServiceTest = func() stateServiceTestType {
	channelServiceMock := &paymentChannelServiceMock{}
	senderAddress := crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	signerPrivateKey := GenerateTestPrivateKey()
	signerAddress := crypto.PubkeyToAddress(signerPrivateKey.PublicKey)

	channelServiceMock.blockchainReader = &BlockchainChannelReader{}

	channelServiceMock.blockchainReader.readChannelFromBlockchain = func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
		return &blockchain.MultiPartyEscrowChannel{
			Recipient: senderAddress,
			Nonce:     big.NewInt(3),
		}, true, nil
	}
	channelServiceMock.blockchainReader.recipientPaymentAddress = func() common.Address {
		return senderAddress
	}

	defaultChannelId := big.NewInt(42)
	defaultSignature, err := hex.DecodeString("0504030201")
	if err != nil {
		panic("Could not make defaultSignature")
	}

	paymentStorage := NewPaymentStorage(NewMemStorage())
	defaultTestChannelData := &PaymentChannelData{
		ChannelID:        defaultChannelId,
		Sender:           senderAddress,
		Signer:           signerAddress,
		Signature:        defaultSignature,
		Nonce:            big.NewInt(3),
		AuthorizedAmount: big.NewInt(12345),
	}

	paymentStorage.Put(getPaymentFromChannel(defaultTestChannelData))
	return stateServiceTestType{
		service: PaymentChannelStateService{
			channelService: channelServiceMock,
			paymentStorage: paymentStorage,
		},
		senderAddress:      senderAddress,
		signerPrivateKey:   signerPrivateKey,
		signerAddress:      signerAddress,
		channelServiceMock: channelServiceMock,

		defaultChannelId:   defaultChannelId,
		defaultChannelKey:  &PaymentChannelKey{ID: defaultChannelId},
		defaultChannelData: defaultTestChannelData,
		defaultRequest: &ChannelStateRequest{
			ChannelId: bigIntToBytes(defaultChannelId),
			Signature: getSignature(bigIntToBytes(defaultChannelId), signerPrivateKey),
		},
		defaultReply: &ChannelStateReply{
			CurrentNonce:        bigIntToBytes(big.NewInt(3)),
			CurrentSignedAmount: bigIntToBytes(big.NewInt(12345)),
			CurrentSignature:    defaultSignature,
		},
	}
}()

func TestGetChannelState(t *testing.T) {
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, err)
	assert.Equal(t, stateServiceTest.defaultReply, reply)
	//Channel's nonce  =  blockchain nonce + 1
	stateServiceTest.defaultChannelData.Nonce = big.NewInt(4)
	reply, err = stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)
	assert.Nil(t, err)
	assert.NotNil(t, reply)

	//reset the channel storage setting to default
	stateServiceTest.defaultChannelData.Nonce = big.NewInt(3)

}

func TestGetChannelStateChannelIdIsNotPaddedByZero(t *testing.T) {
	channelId := big.NewInt(255)
	stateServiceTest.channelServiceMock.Put(&PaymentChannelKey{ID: channelId}, stateServiceTest.defaultChannelData)
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: []byte{0xFF},
			Signature: getSignature(bigIntToBytes(channelId), stateServiceTest.signerPrivateKey),
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, stateServiceTest.defaultReply, reply)
}

func TestGetChannelStateChannelIdIncorrectSignature(t *testing.T) {
	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: bigIntToBytes(stateServiceTest.defaultChannelId),
			Signature: []byte{0x00},
		},
	)

	assert.Equal(t, errors.New("incorrect signature"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateChannelStorageError(t *testing.T) {
	stateServiceTest.channelServiceMock.SetError(errors.New("storage error"))
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(nil, stateServiceTest.defaultRequest)

	assert.Equal(t, errors.New("channel error:storage error"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateChannelNotFound(t *testing.T) {
	channelId := big.NewInt(42)
	stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: bigIntToBytes(channelId),
			Signature: getSignature(bigIntToBytes(channelId), stateServiceTest.signerPrivateKey),
		},
	)

	assert.Equal(t, errors.New("channel is not found, channelId: 42"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateIncorrectSender(t *testing.T) {
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: bigIntToBytes(stateServiceTest.defaultChannelId),
			Signature: getSignature(
				bigIntToBytes(stateServiceTest.defaultChannelId),
				GenerateTestPrivateKey()),
		},
	)

	assert.Equal(t, errors.New("only channel signer can get latest channel state"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateNoOperationsOnThisChannelYet(t *testing.T) {
	channelData := stateServiceTest.defaultChannelData
	channelData.AuthorizedAmount = nil
	channelData.Signature = nil
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		channelData,
	)
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, err)
	expectedReply := stateServiceTest.defaultReply
	expectedReply.CurrentSignedAmount = nil
	expectedReply.CurrentSignature = nil
	expectedReply.OldNonceSignature = nil
	expectedReply.OldNonceSignedAmount = nil
	assert.Equal(t, expectedReply, reply)
}

// Claim tests are already added to escrow_test.go
