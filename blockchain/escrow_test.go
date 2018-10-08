package blockchain

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	result := m.Run()

	os.Exit(result)
}

var testPrivateKey = generatePrivateKey()

func generatePrivateKey() (privateKey *ecdsa.PrivateKey) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Cannot generate private key for test: %v", err))
	}
	return
}

type storageMockType struct {
	data map[channelStorageKey]*PaymentChannelData
}

var storageMock = storageMockType{
	data: make(map[channelStorageKey]*PaymentChannelData),
}

type channelStorageKey string

func getChannelStorageKey(key *PaymentChannelKey) channelStorageKey {
	return channelStorageKey(fmt.Sprintf("%v", key))
}

func (storage *storageMockType) Put(key *PaymentChannelKey, channel *PaymentChannelData) {
	storage.data[getChannelStorageKey(key)] = channel
}

func (storage *storageMockType) Get(key *PaymentChannelKey) (channel *PaymentChannelData, err error) {
	channel, ok := storage.data[getChannelStorageKey(key)]
	if !ok {
		return nil, fmt.Errorf("No value for key: \"%v\"", key)
	}
	return channel, nil
}

func (storage *storageMockType) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) error {
	return nil
}

var processorMock = Processor{}

type amountValidatorMockType struct {
}

var amountValidatorMock = amountValidatorMockType{}

func (amountValidator *amountValidatorMockType) Validate(paymentData *PaymentData) (err *status.Status) {
	return nil
}

func getEscrowMetadata(channelId, channelNonce, amount int) metadata.MD {
	hash := crypto.Keccak256(
		hashPrefix32Bytes,
		crypto.Keccak256(
			processorMock.escrowContractAddress.Bytes(),
			intToUint256(channelId),
			intToUint256(channelNonce),
			intToUint256(amount),
		),
	)

	signature, err := crypto.Sign(hash, testPrivateKey)
	if err != nil {
		panic(fmt.Sprintf("Cannot sign test message: %v", err))
	}

	return metadata.Pairs(
		PaymentChannelIdHeader, strconv.Itoa(channelId),
		PaymentChannelNonceHeader, strconv.Itoa(channelNonce),
		PaymentChannelAmountHeader, strconv.Itoa(amount),
		PaymentChannelSignatureHeader, string(signature))
}

func intToUint256(value int) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(value))
	return common.BytesToHash(bytes).Bytes()
}

func hexToBytes(str string) []byte {
	return common.FromHex(str)
}

func hexToAddress(str string) common.Address {
	return common.Address(common.BytesToAddress(hexToBytes(str)))
}

func TestGetPublicKeyFromPayment(t *testing.T) {
	escrowContractAddress := hexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")
	handler := escrowPaymentHandler{processor: &Processor{escrowContractAddress: escrowContractAddress}}
	payment := paymentData{
		channelKey: &PaymentChannelKey{Id: big.NewInt(1789), Nonce: big.NewInt(1917)},
		amount:     big.NewInt(31415),
		signature:  hexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, hexToAddress("0xc5fdf4076b8f3a5357c5e395ab970b5b54098fef"), *address)
}

func TestValidatePayment(t *testing.T) {
	storageMock.Put(
		&PaymentChannelKey{Id: big.NewInt(42), Nonce: big.NewInt(3)},
		&PaymentChannelData{
			State:            Open,
			Sender:           crypto.PubkeyToAddress(testPrivateKey.PublicKey),
			FullAmount:       big.NewInt(12345),
			Expiration:       time.Now().Add(time.Hour),
			AuthorizedAmount: big.NewInt(12300),
			Signature:        nil,
		},
	)
	md := getEscrowMetadata(42, 3, 12345)
	handler := escrowPaymentHandler{
		md:              md,
		storage:         &storageMock,
		processor:       &processorMock,
		amountValidator: &amountValidatorMock,
	}

	err := handler.validatePayment()

	assert.Nil(t, err)
}
