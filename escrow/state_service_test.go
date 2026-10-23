package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stateServiceTestType struct {
	service            PaymentChannelStateService
	senderAddress      common.Address
	signerPrivateKey   *ecdsa.PrivateKey
	signerAddress      common.Address
	receiverAddress    common.Address
	receiverPvtKy      *ecdsa.PrivateKey
	channelServiceMock *paymentChannelServiceMock
	mpeAddress         common.Address
	defaultChannelId   *big.Int
	defaultChannelKey  *PaymentChannelKey
	defaultChannelData *PaymentChannelData
	defaultRequest     *ChannelStateRequest
	defaultReply       *ChannelStateReply
}

// newStateServiceTest creates independent mocks, storage, and data for each test.
func newStateServiceTest(t *testing.T) stateServiceTestType {
	t.Helper()

	channelServiceMock := &paymentChannelServiceMock{}
	senderAddress := crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	signerPrivateKey := GenerateTestPrivateKey()
	signerAddress := crypto.PubkeyToAddress(signerPrivateKey.PublicKey)
	receiverPvtkey := GenerateTestPrivateKey()
	receiverAddress := crypto.PubkeyToAddress(receiverPvtkey.PublicKey)

	channelServiceMock.blockchainReader = &BlockchainChannelReader{}
	ethereumBlock := big.NewInt(53)
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
	require.NoError(t, err)

	paymentStorage := NewPaymentStorage(storage.NewMemStorage())
	mpeAddress := common.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")

	return stateServiceTestType{
		service: PaymentChannelStateService{
			channelService:               channelServiceMock,
			paymentStorage:               paymentStorage,
			mpeAddress:                   func() common.Address { return mpeAddress },
			compareWithLatestBlockNumber: func(*big.Int) error { return nil },
		},
		senderAddress:      senderAddress,
		receiverAddress:    receiverAddress,
		receiverPvtKy:      receiverPvtkey,
		signerPrivateKey:   signerPrivateKey,
		signerAddress:      signerAddress,
		channelServiceMock: channelServiceMock,
		mpeAddress:         mpeAddress,

		defaultChannelId:  defaultChannelId,
		defaultChannelKey: &PaymentChannelKey{ID: defaultChannelId},
		defaultChannelData: &PaymentChannelData{
			ChannelID:        defaultChannelId,
			Sender:           senderAddress,
			Signer:           signerAddress,
			Recipient:        receiverAddress,
			Signature:        defaultSignature,
			Nonce:            big.NewInt(3),
			AuthorizedAmount: big.NewInt(12345),
		},
		defaultRequest: &ChannelStateRequest{
			CurrentBlock: ethereumBlock.Uint64(),
			ChannelId:    bigIntToBytes(defaultChannelId),
			Signature: getSignature(
				getChannelStateRequestMessage(mpeAddress, defaultChannelId, ethereumBlock),
				signerPrivateKey,
			),
		},
		defaultReply: &ChannelStateReply{
			CurrentNonce:        bigIntToBytes(big.NewInt(3)),
			CurrentSignedAmount: bigIntToBytes(big.NewInt(12345)),
			CurrentSignature:    bytes.Clone(defaultSignature),
		},
	}
}

func getChannelStateRequestMessage(mpeAddress common.Address, channelId *big.Int, currentBlock *big.Int) []byte {
	message := bytes.Join([][]byte{
		[]byte("__get_channel_state"),
		mpeAddress.Bytes(),
		bigIntToBytes(channelId),
		math.U256Bytes(currentBlock),
	}, nil)
	return message
}

func TestStateServiceFixtureIsolation(t *testing.T) {
	first := newStateServiceTest(t)
	second := newStateServiceTest(t)

	first.channelServiceMock.Put(first.defaultChannelKey, first.defaultChannelData)
	require.NoError(t, first.service.paymentStorage.Put(getPaymentFromChannel(first.defaultChannelData)))
	first.channelServiceMock.SetError(errors.New("storage error"))
	first.channelServiceMock.blockchainReader.readChannelFromBlockchain = func(*big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
		return nil, false, errors.New("blockchain error")
	}
	first.defaultChannelData.AuthorizedAmount.SetInt64(0)
	first.defaultChannelData.Signature[0] = 0
	first.defaultChannelData.AuthorizedAmount = nil
	first.defaultChannelData.Signature = nil
	first.defaultRequest.Signature[0] ^= 0xff
	first.defaultReply.CurrentSignedAmount[0] = 0xff
	first.defaultReply.CurrentSignature = nil

	_, found, err := second.channelServiceMock.PaymentChannel(second.defaultChannelKey)
	require.NoError(t, err)
	require.False(t, found)
	payments, err := second.service.paymentStorage.GetAll()
	require.NoError(t, err)
	require.Empty(t, payments)

	second.channelServiceMock.Put(second.defaultChannelKey, second.defaultChannelData)
	reply, err := second.service.GetChannelState(t.Context(), second.defaultRequest)
	require.NoError(t, err)
	expected := &ChannelStateReply{
		CurrentNonce:        bigIntToBytes(big.NewInt(3)),
		CurrentSignedAmount: bigIntToBytes(big.NewInt(12345)),
		CurrentSignature:    []byte{0x05, 0x04, 0x03, 0x02, 0x01},
	}
	assert.Equal(t, expected, reply)
	assert.Equal(t, expected, second.defaultReply)
}

func TestGetChannelState(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	require.NoError(t, err)
	assert.Equal(t, stateServiceTest.defaultReply, reply)

}

func TestGetChannelStateWhenNonceDiffers(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	previousSignature, err := hex.DecodeString("0708090A0B")
	require.NoError(t, err)
	previousChannelData := &PaymentChannelData{
		ChannelID:        stateServiceTest.defaultChannelId,
		Sender:           stateServiceTest.senderAddress,
		Signer:           stateServiceTest.signerAddress,
		Recipient:        stateServiceTest.receiverAddress,
		Signature:        previousSignature,
		Nonce:            big.NewInt(2),
		AuthorizedAmount: big.NewInt(123),
	}
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	payment := getPaymentFromChannel(previousChannelData)
	require.NoError(t, stateServiceTest.service.paymentStorage.Put(payment))
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain = func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
		mpeChannel := &blockchain.MultiPartyEscrowChannel{
			Recipient: stateServiceTest.senderAddress,
			Nonce:     big.NewInt(2),
		}
		return mpeChannel, true, nil
	}

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	require.NoError(t, err)
	require.NotNil(t, reply)
	assert.Equal(t, bigIntToBytes(big.NewInt(3)), reply.CurrentNonce)
	assert.Equal(t, stateServiceTest.defaultChannelData.Signature, reply.CurrentSignature)
	assert.Equal(t, bigIntToBytes(big.NewInt(12345)), reply.CurrentSignedAmount)
	assert.Equal(t, bigIntToBytes(big.NewInt(123)), reply.OldNonceSignedAmount)
	assert.Equal(t, previousChannelData.Signature, reply.OldNonceSignature)
}

func TestGetChannelStateWhenReceiverMakesRequest(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)

	_, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			CurrentBlock: big.NewInt(53).Uint64(),
			ChannelId:    bigIntToBytes(stateServiceTest.defaultChannelId),
			Signature: getSignature(
				getChannelStateRequestMessage(stateServiceTest.mpeAddress, stateServiceTest.defaultChannelId, big.NewInt(53)),
				stateServiceTest.receiverPvtKy,
			),
		},
	)

	require.NoError(t, err)

}

func TestGetChannelStateChannelIdIncorrectSignature(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
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
	stateServiceTest := newStateServiceTest(t)
	stateServiceTest.channelServiceMock.SetError(errors.New("storage error"))

	reply, err := stateServiceTest.service.GetChannelState(t.Context(), stateServiceTest.defaultRequest)

	assert.Equal(t, errors.New("channel error:storage error"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateChannelNotFound(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	channelId := big.NewInt(42)

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
	stateServiceTest := newStateServiceTest(t)
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: bigIntToBytes(stateServiceTest.defaultChannelId),
			Signature: getSignature(
				bigIntToBytes(stateServiceTest.defaultChannelId),
				GenerateTestPrivateKey()),
		},
	)

	assert.Equal(t, errors.New("only channel signer/sender/receiver can get latest channel state"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateNoOperationsOnThisChannelYet(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	channelData := stateServiceTest.defaultChannelData
	channelData.AuthorizedAmount = nil
	channelData.Signature = nil
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		channelData,
	)

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	require.NoError(t, err)
	expectedReply := stateServiceTest.defaultReply
	expectedReply.CurrentSignedAmount = nil
	expectedReply.CurrentSignature = nil
	expectedReply.OldNonceSignature = nil
	expectedReply.OldNonceSignedAmount = nil
	assert.Equal(t, expectedReply, reply)
}

func TestGetChannelStateBlockchainError(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain =
		func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, errors.New("Test error from blockchain reads")
		}

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, reply)
	assert.Equal(t, errors.New("channel error:Test error from blockchain reads"), err)
}

func TestGetChannelStateNoChannelInBlockchain(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	stateServiceTest.channelServiceMock.blockchainReader.readChannelFromBlockchain =
		func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
			return nil, false, nil
		}

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, reply)
	assert.Equal(t, errors.New("unable to read channel details from blockchain."), err)
}

func TestGetChannelStateNonceIncrementedInBlockchainNoOldPayment(t *testing.T) {
	stateServiceTest := newStateServiceTest(t)
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

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)

	assert.Nil(t, reply)
	assert.Equal(t, errors.New("channel has different nonce in local storage and blockchain and old payment is not found in storage"), err)
}
