package blockchain

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
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

func TestValidatePayment(t *testing.T) {
	t.Skip("Not implemented yet")

	storageMock.Put(
		&PaymentChannelKey{big.NewInt(42), big.NewInt(3)},
		&PaymentChannelData{
			Open,
			crypto.PubkeyToAddress(testPrivateKey.PublicKey),
			big.NewInt(12345),
			time.Now().Add(time.Hour),
			big.NewInt(12300),
			nil,
		},
	)
	md := getEscrowMetadata(42, 3, 12345)
	handler := escrowPaymentHandler{md, &storageMock, &processorMock}

	err := handler.validatePayment()

	assert.Nil(t, err)
}
