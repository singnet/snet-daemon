package escrow

import (
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type stateServiceTestType struct {
	service          PaymentChannelStateService
	senderPrivateKey *ecdsa.PrivateKey
	senderAddress    common.Address
	storageMock      *storageMockType

	defaultChannelId   *big.Int
	defaultChannelKey  *PaymentChannelKey
	defaultChannelData *PaymentChannelData
	defaultRequest     *ChannelStateRequest
	defaultReply       *ChannelStateReply
}

var stateServiceTest = func() stateServiceTestType {
	storageMock := &storageMockType{
		delegate: NewPaymentChannelStorage(NewMemStorage()),
		err:      nil,
	}
	senderPrivateKey := generatePrivateKey()
	senderAddress := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)

	defaultChannelId := big.NewInt(42)
	defaultSignature, err := hex.DecodeString("0504030201")
	if err != nil {
		panic("Could not make defaultSignature")
	}

	return stateServiceTestType{
		service: PaymentChannelStateService{
			latest: storageMock,
		},
		senderPrivateKey: senderPrivateKey,
		senderAddress:    senderAddress,
		storageMock:      storageMock,

		defaultChannelId:  defaultChannelId,
		defaultChannelKey: &PaymentChannelKey{ID: defaultChannelId},
		defaultChannelData: &PaymentChannelData{
			Sender:           senderAddress,
			Signature:        defaultSignature,
			Nonce:            big.NewInt(3),
			AuthorizedAmount: big.NewInt(12345),
		},
		defaultReply: &ChannelStateReply{
			CurrentNonce:        bigIntToBytes(big.NewInt(3)),
			CurrentSignedAmount: bigIntToBytes(big.NewInt(12345)),
			CurrentSignature:    defaultSignature,
		},
	}
}()

func TestGetChannelState(t *testing.T) {
	channelId := big.NewInt(42)
	signature, err := hex.DecodeString("0504030201")
	assert.Nil(t, err)
	stateServiceTest.storageMock.Put(
		&PaymentChannelKey{ID: channelId},
		&PaymentChannelData{
			Sender:           stateServiceTest.senderAddress,
			Signature:        signature,
			Nonce:            big.NewInt(3),
			AuthorizedAmount: big.NewInt(12345),
		},
	)
	defer stateServiceTest.storageMock.Clear()

	reply, err := stateServiceTest.service.GetChannelState(
		nil,
		&ChannelStateRequest{
			ChannelId: bigIntToBytes(channelId),
			Signature: getSignature(bigIntToBytes(channelId), stateServiceTest.senderPrivateKey),
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, &ChannelStateReply{
		CurrentNonce:        bigIntToBytes(big.NewInt(3)),
		CurrentSignedAmount: bigIntToBytes(big.NewInt(12345)),
		CurrentSignature:    signature,
	}, reply)
}

func TestGetChannelStateChannelIdIsNotPaddedByZero(t *testing.T) {
	channelId := big.NewInt(255)
	stateServiceTest.storageMock.Put(&PaymentChannelKey{ID: channelId}, stateServiceTest.defaultChannelData)
	defer stateServiceTest.storageMock.Clear()

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
