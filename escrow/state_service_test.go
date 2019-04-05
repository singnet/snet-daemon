package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/authutils"
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

	defaultChannelId   *big.Int
	defaultChannelKey  *PaymentChannelKey
	defaultChannelData *PaymentChannelData
	defaultRequest     *ChannelStateRequest
	defaultReply       *ChannelStateReply

	defaultChannelIdA   *big.Int
	defaultChannelKeyA  *PaymentChannelKey
	defaultChannelDataA *PaymentChannelData
	defaultRequestA 	*ChannelStateRequest
	defaultReplyA 		*ChannelStateReply
}

var stateServiceTest = func() stateServiceTestType {
	channelServiceMock := &paymentChannelServiceMock{}
	senderAddress := crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	signerPrivateKey := GenerateTestPrivateKey()
	signerAddress := crypto.PubkeyToAddress(signerPrivateKey.PublicKey)

	defaultChannelId := big.NewInt(42)
	defaultSignature, err := hex.DecodeString("0504030201")

	if err != nil {
		panic("Could not make default Signature")
	}

	defaultPrevSignature, err := hex.DecodeString("0403020100")
	if err != nil {
		panic("Could not make default previous Signature")
	}

	//abi.U256(big.NewInt(int64(request.CurrentBlock))),
	defaultBlock, err := authutils.CurrentBlock()
	if err != nil {
		panic("Could not read current blocknumber")
	}

	defaultChannelIdA := big.NewInt(43)
	message := bytes.Join([][]byte{
		[]byte ("__get_channel_state"),
		defaultChannelIdA.Bytes(),
		abi.U256(defaultBlock),
	}, nil)
	newFormatSignature := getSignature(message, signerPrivateKey)

	return stateServiceTestType{
		service: PaymentChannelStateService{
			channelService: channelServiceMock,
		},
		senderAddress:      senderAddress,
		signerPrivateKey:   signerPrivateKey,
		signerAddress:      signerAddress,
		channelServiceMock: channelServiceMock,

		defaultChannelId:  defaultChannelId,
		defaultChannelKey: &PaymentChannelKey{ID: defaultChannelId},
		defaultChannelData: &PaymentChannelData{
			ChannelID:            defaultChannelId,
			Sender:               senderAddress,
			Signer:               signerAddress,
			Signature:            defaultSignature,
			Nonce:                big.NewInt(3),
			AuthorizedAmount:     big.NewInt(12345),
			OldNonceSignature:    defaultPrevSignature,
			OldNonceSignedAmount: big.NewInt(2345),
		},
		defaultRequest: &ChannelStateRequest{
			ChannelId: bigIntToBytes(defaultChannelId),
			CurrentBlock: defaultBlock.Uint64(),
			Signature: getSignature(bigIntToBytes(defaultChannelId), signerPrivateKey),
		},
		defaultReply: &ChannelStateReply{
			CurrentNonce:         bigIntToBytes(big.NewInt(3)),
			CurrentSignedAmount:  bigIntToBytes(big.NewInt(12345)),
			CurrentSignature:     defaultSignature,
			OldNonceSignature:    defaultPrevSignature,
			OldNonceSignedAmount: bigIntToBytes(big.NewInt(2345)),
		},
		defaultChannelIdA:  defaultChannelIdA,
		defaultChannelKeyA: &PaymentChannelKey{ID: defaultChannelIdA},
		defaultChannelDataA: &PaymentChannelData{
			ChannelID:            defaultChannelIdA,
			Sender:               senderAddress,
			Signer:               signerAddress,
			Signature:            defaultSignature,
			Nonce:                big.NewInt(7),
			AuthorizedAmount:     big.NewInt(8345),
			OldNonceSignature:    defaultPrevSignature,
			OldNonceSignedAmount: big.NewInt(2323),
		},
		defaultRequestA: &ChannelStateRequest{
			ChannelId: bigIntToBytes(defaultChannelIdA),
			CurrentBlock: defaultBlock.Uint64(),
			Signature: newFormatSignature,
		},
		defaultReplyA: &ChannelStateReply{
			CurrentNonce:         bigIntToBytes(big.NewInt(7)),
			CurrentSignedAmount:  bigIntToBytes(big.NewInt(8345)),
			CurrentSignature:     defaultSignature,
			OldNonceSignature:    defaultPrevSignature,
			OldNonceSignedAmount: bigIntToBytes(big.NewInt(2323)),
		},
	}
}()

func TestGetChannelStateWithNewSignatureFormat(t *testing.T) {
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKeyA,
		stateServiceTest.defaultChannelDataA,
	)
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequestA,
	)

	assert.Nil(t, err)
	assert.Equal(t, stateServiceTest.defaultReplyA, reply)
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
	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		stateServiceTest.defaultChannelData,
	)
	defer stateServiceTest.channelServiceMock.Clear()

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

func TestGetChannelStateVerifyPrevChannelState(t *testing.T) {
	tmpSignature, err := hex.DecodeString("0102030405")
	tmpSignedAmount := big.NewInt(34545)

	channelData := stateServiceTest.defaultChannelData
	channelData.OldNonceSignature = tmpSignature
	channelData.OldNonceSignedAmount = tmpSignedAmount

	stateServiceTest.channelServiceMock.Put(
		stateServiceTest.defaultChannelKey,
		channelData,
	)
	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		stateServiceTest.defaultRequest,
	)
	assert.Nil(t, err)
	expectedReply := stateServiceTest.defaultReply
	expectedReply.OldNonceSignature = tmpSignature
	expectedReply.OldNonceSignedAmount = bigIntToBytes(tmpSignedAmount)
	assert.Equal(t, expectedReply, reply)

	defer stateServiceTest.channelServiceMock.Clear()
}

// prev signature and authorized values  could be nil
func TestGetChannelStateVerifyPrevChannelStateBegining(t *testing.T) {
	channelData := stateServiceTest.defaultChannelData
	channelData.OldNonceSignature = nil
	channelData.OldNonceSignedAmount = big.NewInt(-1)

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
	expectedReply.OldNonceSignature = nil
	expectedReply.OldNonceSignedAmount = bigIntToBytes(big.NewInt(-1))
	assert.Equal(t, expectedReply, reply)
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
