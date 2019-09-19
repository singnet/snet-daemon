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
	defaultChannelId := big.NewInt(42)
	channelServiceMock.blockchainReader.readChannelFromBlockchain = func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
		mpeChannel := &blockchain.MultiPartyEscrowChannel{
			Recipient: senderAddress,
			Nonce:     big.NewInt(3),
		}
		return mpeChannel, true, nil
	}

	channelServiceMock.blockchainReader.recipientPaymentAddress = func() common.Address {
		return senderAddress
	}

	defaultSignature, err := hex.DecodeString("0504030201")
	if err != nil {
		panic("Could not make defaultSignature")
	}

	paymentStorage := NewPaymentStorage(NewMemStorage())
	verificationAddress:= common.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")

	return stateServiceTestType{
		service: PaymentChannelStateService{
			channelService: channelServiceMock,
			paymentStorage: paymentStorage,
			mpeAddress: func() common.Address {return verificationAddress},
		},
		senderAddress:      senderAddress,
		signerPrivateKey:   signerPrivateKey,
		signerAddress:      signerAddress,
		channelServiceMock: channelServiceMock,

		defaultChannelId:  defaultChannelId,
		defaultChannelKey: &PaymentChannelKey{ID: defaultChannelId},
		defaultChannelData: &PaymentChannelData{
			ChannelID:        defaultChannelId,
			Sender:           senderAddress,
			Signer:           signerAddress,
			Signature:        defaultSignature,
			Nonce:            big.NewInt(3),
			AuthorizedAmount: big.NewInt(12345),
			FullAmount:big.NewInt(123789),
			Expiration:big.NewInt(1234444444444),
		},
		defaultRequest: &ChannelStateRequest{
			ChannelId: bigIntToBytes(defaultChannelId),
			Signature: getSignature(bigIntToBytes(defaultChannelId), signerPrivateKey),
		},
		defaultReply: &ChannelStateReply{
			CurrentNonce:        bigIntToBytes(big.NewInt(3)),
			CurrentSignedAmount: bigIntToBytes(big.NewInt(12345)),
			CurrentSignature:    defaultSignature,
			AmountDeposited:  bigIntToBytes(big.NewInt(123789)),
			Expiry: bigIntToBytes(big.NewInt(1234444444444)),
			ChannelId:bigIntToBytes(big.NewInt(42)),

		},
	}
}()

func cleanup() {
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain = func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
		mpeChannel := &blockchain.MultiPartyEscrowChannel{
			Recipient: stateServiceTest.senderAddress,
			Nonce:     big.NewInt(3),
		}
		return mpeChannel, true, nil
	}
	paymentStorage := NewPaymentStorage(NewMemStorage())
	stateServiceTest.service.paymentStorage = paymentStorage
	stateServiceTest.channelServiceMock.Clear()
}

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

}

func TestGetChannelStateWhenNonceDiffers(t *testing.T) {
	previousSignature, _ := hex.DecodeString("0708090A0B")
	previousChannelData := &PaymentChannelData{
		ChannelID:        stateServiceTest.defaultChannelId,
		Sender:           stateServiceTest.senderAddress,
		Signer:           stateServiceTest.signerAddress,
		Signature:        previousSignature,
		Nonce:            big.NewInt(2),
		AuthorizedAmount: big.NewInt(123),
		Expiration:big.NewInt(1234444444444),
	}
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	payment := getPaymentFromChannel(previousChannelData)
	stateServiceTest.service.paymentStorage.Put(payment)
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain = func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
		mpeChannel := &blockchain.MultiPartyEscrowChannel{
			Recipient: stateServiceTest.senderAddress,
			Nonce:     big.NewInt(2),
		}
		return mpeChannel, true, nil
	}
	defer cleanup()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, err)
	assert.Equal(t, bigIntToBytes(big.NewInt(3)), reply.CurrentNonce)
	assert.Equal(t, stateServiceTest.defaultChannelData.Signature, reply.CurrentSignature)
	assert.Equal(t, bigIntToBytes(big.NewInt(12345)), reply.CurrentSignedAmount)
	assert.Equal(t, bigIntToBytes(big.NewInt(123)), reply.OldNonceSignedAmount)
	assert.Equal(t, previousChannelData.Signature, reply.OldNonceSignature)
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
	expectedReply.CurrentSignedAmount = bigIntToBytes(big.NewInt(0))
	expectedReply.CurrentSignature = nil
	expectedReply.OldNonceSignature = nil
	expectedReply.OldNonceSignedAmount = nil
	assert.Equal(t, expectedReply, reply)
}

func TestGetChannelStateBlockchainError(t *testing.T) {
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain =
		func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, errors.New("Test error from blockchain reads")
		}
	defer cleanup()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, reply)
	assert.Equal(t, errors.New("channel error:Test error from blockchain reads"), err)
}

func TestGetChannelStateNoChannelInBlockchain(t *testing.T) {
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain =
		func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, nil
		}
	defer cleanup()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, reply)
	assert.Equal(t, errors.New("unable to read channel details from blockchain."), err)
}

func TestGetChannelStateNonceIncrementedInBlockchainNoOldPayment(t *testing.T) {
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	blockchainChannelData := &blockchain.MultiPartyEscrowChannel{
		Recipient: stateServiceTest.senderAddress,
		Nonce:     big.NewInt(0).Sub(stateServiceTest.defaultChannelData.Nonce, big.NewInt(1)),
	}
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain =
		func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return blockchainChannelData, true, nil
		}
	defer cleanup()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, reply)
	assert.Equal(t, errors.New("channel has different nonce in local storage and blockchain and old payment is not found in storage"), err)
}

// Claim tests are already added to escrow_test.go