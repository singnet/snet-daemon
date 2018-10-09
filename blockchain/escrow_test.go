package blockchain

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
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
var testPublicKey = crypto.PubkeyToAddress(testPrivateKey.PublicKey)

func generatePrivateKey() (privateKey *ecdsa.PrivateKey) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Cannot generate private key for test: %v", err))
	}
	return
}

type channelStorageKey string

type storageMockType struct {
	data map[channelStorageKey]*PaymentChannelData
}

var storageMock = storageMockType{
	data: make(map[channelStorageKey]*PaymentChannelData),
}

func getChannelStorageKey(key *PaymentChannelKey) channelStorageKey {
	return channelStorageKey(fmt.Sprintf("%v", key))
}

func (storage *storageMockType) Put(key *PaymentChannelKey, channel *PaymentChannelData) (err error) {
	storage.data[getChannelStorageKey(key)] = channel
	return nil
}

func (storage *storageMockType) Get(key *PaymentChannelKey) (channel *PaymentChannelData, err error) {
	channel, ok := storage.data[getChannelStorageKey(key)]
	if !ok {
		return nil, fmt.Errorf("No value for key: \"%v\"", key)
	}
	return channel, nil
}

func (storage *storageMockType) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (err error) {
	current, err := storage.Get(key)
	if err != nil {
		return
	}
	if toJSON(current) != toJSON(prevState) {
		return fmt.Errorf("Current state is not equal to expected, current: %v, expected: %v", current, prevState)
	}
	return storage.Put(key, newState)
}

func (storage *storageMockType) Clear() {
	storage.data = make(map[channelStorageKey]*PaymentChannelData)
}

var processorMock = Processor{
	escrowContractAddress: hexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf"),
}

type incomeValidatorMockType struct {
	err *status.Status
}

var incomeValidatorMock = incomeValidatorMockType{}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeData) (err *status.Status) {
	return incomeValidator.err
}

var paymentHandler = escrowPaymentHandler{
	storage:         &storageMock,
	processor:       &processorMock,
	incomeValidator: &incomeValidatorMock,
}

func getSignature(contractAddress *common.Address, channelID, channelNonce, amount int64) (signature []byte) {
	hash := crypto.Keccak256(
		hashPrefix32Bytes,
		crypto.Keccak256(
			processorMock.escrowContractAddress.Bytes(),
			intToUint256(channelID),
			intToUint256(channelNonce),
			intToUint256(amount),
		),
	)

	signature, err := crypto.Sign(hash, testPrivateKey)
	if err != nil {
		panic(fmt.Sprintf("Cannot sign test message: %v", err))
	}

	return signature
}

func intToUint256(value int64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(value))
	return common.BytesToHash(bytes).Bytes()
}

func getEscrowMetadata(channelID, channelNonce, amount int64) metadata.MD {
	md := metadata.New(map[string]string{})
	if channelID != 0 {
		md.Set(PaymentChannelIDHeader, strconv.FormatInt(channelID, 10))
	}
	if channelNonce != 0 {
		md.Set(PaymentChannelNonceHeader, strconv.FormatInt(channelNonce, 10))
	}
	if amount != 0 {
		md.Set(PaymentChannelAmountHeader, strconv.FormatInt(amount, 10))
	}
	md.Set(PaymentChannelSignatureHeader, string(getSignature(&processorMock.escrowContractAddress, channelID, channelNonce, amount)))
	return md
}

type testPaymentData struct {
	channelID, channelNonce, fullAmount, prevAmount, newAmount int64
	state                                                      PaymentChannelState
	expiration                                                 time.Time
	signature                                                  []byte
}

func newPaymentChannelKey(ID, nonce int64) *PaymentChannelKey {
	return &PaymentChannelKey{ID: big.NewInt(ID), Nonce: big.NewInt(nonce)}
}

func getTestPayment(data *testPaymentData) *escrowPaymentType {
	signature := data.signature
	if signature == nil {
		signature = getSignature(&processorMock.escrowContractAddress, data.channelID, data.channelNonce, data.newAmount)
	}
	return &escrowPaymentType{
		grpcContext: &GrpcStreamContext{MD: getEscrowMetadata(data.channelID, data.channelNonce, data.newAmount)},
		channelKey:  newPaymentChannelKey(data.channelID, data.channelNonce),
		amount:      big.NewInt(data.newAmount),
		signature:   signature,
		channel: &PaymentChannelData{
			State:            data.state,
			Sender:           testPublicKey,
			FullAmount:       big.NewInt(data.fullAmount),
			Expiration:       data.expiration,
			AuthorizedAmount: big.NewInt(data.prevAmount),
			Signature:        nil,
		},
	}
}

func getTestContext(data *testPaymentData) *GrpcStreamContext {
	storageMock.Put(
		newPaymentChannelKey(data.channelID, data.channelNonce),
		&PaymentChannelData{
			State:            data.state,
			Sender:           testPublicKey,
			FullAmount:       big.NewInt(data.fullAmount),
			Expiration:       data.expiration,
			AuthorizedAmount: big.NewInt(data.prevAmount),
			Signature:        nil,
		},
	)
	md := getEscrowMetadata(data.channelID, data.channelNonce, data.newAmount)
	return &GrpcStreamContext{
		MD: md,
	}
}

func clearTestContext() {
	storageMock.Clear()
}

func pairToString(data []byte, err error) string {
	if err != nil {
		panic(fmt.Sprintf("Unexpected error: %v", err))
	}
	return string(data)
}

func toJSON(data interface{}) string {
	return pairToString(json.Marshal(data))
}

func TestGetPublicKeyFromPayment(t *testing.T) {
	handler := escrowPaymentHandler{
		processor: &Processor{escrowContractAddress: hexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")},
	}
	payment := escrowPaymentType{
		channelKey: newPaymentChannelKey(1789, 1917),
		amount:     big.NewInt(31415),
		// message hash: 04cc38aa4a27976907ef7382182bc549957dc9d2e21eb73651ad6588d5cd4d8f
		signature: hexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, hexToAddress("0xc5fdf4076b8f3a5357c5e395ab970b5b54098fef"), *address)
}

func TestGetPublicKeyFromPayment2(t *testing.T) {
	handler := escrowPaymentHandler{
		processor: &Processor{escrowContractAddress: hexToAddress("0x39ee715b50e78a920120c1ded58b1a47f571ab75")},
	}
	payment := escrowPaymentType{
		channelKey: newPaymentChannelKey(1789, 1917),
		amount:     big.NewInt(31415),
		signature:  hexToBytes("0xde4e998341307b036e460b1cc1593ddefe2e9ea261bd6c3d75967b29b2c3d0a24969b4a32b099ae2eded90bbc213ad0a159a66af6d55be7e04f724ffa52ce3cc1b"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, hexToAddress("0x592E3C0f3B038A0D673F19a18a773F993d4b2610"), *address)
}

func TestPaymentChannelToJSON(t *testing.T) {
	channel := PaymentChannelData{
		State:            Open,
		Sender:           testPublicKey,
		FullAmount:       big.NewInt(12345),
		Expiration:       time.Now().Add(time.Hour),
		AuthorizedAmount: big.NewInt(12300),
		Signature:        hexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	bytes, err := json.Marshal(channel)
	assert.Nil(t, err)

	channelCopy := &PaymentChannelData{}
	err = json.Unmarshal(bytes, channelCopy)
	assert.Nil(t, err)
}

func TestGetPayment(t *testing.T) {
	data := &testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	}
	context := getTestContext(data)
	defer clearTestContext()

	payment, err := paymentHandler.Payment(context)
	assert.Nil(t, err)
	assert.Equal(t, toJSON(getTestPayment(data)), toJSON(payment))
}

func TestGetPaymentNoChannelId(t *testing.T) {
	context := getTestContext(&testPaymentData{
		channelID:    0,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	})
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
}

func TestGetPaymentNoChannelNonce(t *testing.T) {
	context := getTestContext(&testPaymentData{
		channelID:    42,
		channelNonce: 0,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	})
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-nonce\""), err)
}

func TestGetPaymentNoChannelAmount(t *testing.T) {
	context := getTestContext(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    0,
		prevAmount:   12300,
		state:        Open,
	})
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-amount\""), err)
}

func TestGetPaymentNoChannel(t *testing.T) {
	context := getTestContext(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	})
	storageMock.Clear()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "payment channel \"{ID: 42, Nonce: 3}\" not found"), err)
}

func TestValidatePayment(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	})

	err := paymentHandler.Validate(payment)

	assert.Nil(t, err)
}

func TestValidatePaymentChannelIsNotOpen(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Closed,
	})

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel \"{ID: 42, Nonce: 3}\" is not opened"), err)
}

func TestValidatePaymentIncorrectSignature(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
		signature:    hexToBytes("0x0000"),
	})

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
}

func TestValidatePaymentIncorrectSigner(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
		signature:    hexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	})

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment is not signed by channel sender"), err)
}

func TestValidatePaymentExpiredChannel(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	})

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel is expired since \"2009-11-10 23:00:00 +0000 UTC\""), err)
}

func TestValidatePaymentAmountIsTooBig(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12346,
		prevAmount:   12300,
		state:        Open,
	})

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: 12345, payment amount: 12346"), err)
}

func TestValidatePaymentIncorrectIncome(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		channelID:    42,
		channelNonce: 3,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12345,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	})
	incomeErr := status.New(codes.Unauthenticated, "incorrect payment income: \"45\", expected \"46\"")
	paymentHandler := escrowPaymentHandler{
		storage:         &storageMock,
		processor:       &processorMock,
		incomeValidator: &incomeValidatorMockType{err: incomeErr},
	}

	err := paymentHandler.Validate(payment)

	assert.Equal(t, incomeErr, err)
}

func TestCompletePayment(t *testing.T) {
	data := &testPaymentData{
		channelID:    43,
		channelNonce: 4,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12346,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	}
	getTestContext(data)
	payment := getTestPayment(data)

	err := paymentHandler.Complete(payment)
	channelState, e := storageMock.Get(newPaymentChannelKey(43, 4))

	assert.Nil(t, err)
	assert.Nil(t, e)
	assert.Equal(t, toJSON(&PaymentChannelData{
		State:            Open,
		Sender:           testPublicKey,
		FullAmount:       big.NewInt(12346),
		Expiration:       payment.channel.Expiration,
		AuthorizedAmount: big.NewInt(12345),
		Signature:        payment.signature,
	}), toJSON(channelState))
}

func TestCompletePaymentCannotUpdateChannel(t *testing.T) {
	data := &testPaymentData{
		channelID:    43,
		channelNonce: 4,
		expiration:   time.Now().Add(time.Hour),
		fullAmount:   12346,
		newAmount:    12345,
		prevAmount:   12300,
		state:        Open,
	}
	payment := getTestPayment(data)

	err := paymentHandler.Complete(payment)

	assert.Equal(t, status.New(codes.Internal, "unable to store new payment channel state"), err)
}
