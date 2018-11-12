package escrow

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type stateServiceTestType struct {
	service            PaymentChannelStateService
	senderPrivateKey   *ecdsa.PrivateKey
	senderAddress      common.Address
	channelServiceMock *paymentChannelServiceMock

	defaultChannelId   *big.Int
	defaultChannelKey  *PaymentChannelKey
	defaultChannelData *PaymentChannelData
	defaultRequest     *ChannelStateRequest
	defaultReply       *ChannelStateReply
}

type paymentChannelServiceMock struct {
	paymentChannelService

	err  error
	key  *PaymentChannelKey
	data *PaymentChannelData
}

func (p *paymentChannelServiceMock) PaymentChannel(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
	if p.err != nil {
		return nil, false, p.err
	}
	if p.key == nil || p.key.ID.Cmp(key.ID) != 0 {
		return nil, false, nil
	}
	return p.data, true, nil
}

func (p *paymentChannelServiceMock) Put(key *PaymentChannelKey, data *PaymentChannelData) {
	p.key = key
	p.data = data
}

func (p *paymentChannelServiceMock) SetError(err error) {
	p.err = err
}

func (p *paymentChannelServiceMock) Clear() {
	p.key = nil
	p.data = nil
	p.err = nil
}

var stateServiceTest = func() stateServiceTestType {
	channelServiceMock := &paymentChannelServiceMock{}
	senderPrivateKey := generatePrivateKey()
	senderAddress := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)

	defaultChannelId := big.NewInt(42)
	defaultSignature, err := hex.DecodeString("0504030201")
	if err != nil {
		panic("Could not make defaultSignature")
	}

	return stateServiceTestType{
		service: PaymentChannelStateService{
			channelService: channelServiceMock,
		},
		senderPrivateKey:   senderPrivateKey,
		senderAddress:      senderAddress,
		channelServiceMock: channelServiceMock,

		defaultChannelId:  defaultChannelId,
		defaultChannelKey: &PaymentChannelKey{ID: defaultChannelId},
		defaultChannelData: &PaymentChannelData{
			Sender:           senderAddress,
			Signature:        defaultSignature,
			Nonce:            big.NewInt(3),
			AuthorizedAmount: big.NewInt(12345),
		},
		defaultRequest: &ChannelStateRequest{
			ChannelId: bigIntToBytes(defaultChannelId),
			Signature: getSignature(bigIntToBytes(defaultChannelId), senderPrivateKey),
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
}

func TestGetChannelStateChannelIdIsNotPaddedByZero(t *testing.T) {
	channelId := big.NewInt(255)
	stateServiceTest.channelServiceMock.Put(&PaymentChannelKey{ID: channelId}, stateServiceTest.defaultChannelData)
	defer stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: []byte{0xFF},
			Signature: getSignature(bigIntToBytes(channelId), stateServiceTest.senderPrivateKey),
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

	assert.Equal(t, errors.New("channel storage error"), err)
	assert.Nil(t, reply)
}

func TestGetChannelStateChannelNotFound(t *testing.T) {
	channelId := big.NewInt(42)
	stateServiceTest.channelServiceMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: bigIntToBytes(channelId),
			Signature: getSignature(bigIntToBytes(channelId), stateServiceTest.senderPrivateKey),
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
				generatePrivateKey()),
		},
	)

	assert.Equal(t, errors.New("only channel sender can get latest channel state"), err)
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
	assert.Equal(t, expectedReply, reply)
}
