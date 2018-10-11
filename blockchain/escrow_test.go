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
	escrowContractAddress: HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf"),
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
		HashPrefix32Bytes,
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
	ChannelID, ChannelNonce, FullAmount, PrevAmount, NewAmount int64
	State                                                      PaymentChannelState
	Expiration                                                 time.Time
	Signature                                                  []byte
}

func newPaymentChannelKey(ID, nonce int64) *PaymentChannelKey {
	return &PaymentChannelKey{ID: big.NewInt(ID), Nonce: big.NewInt(nonce)}
}

var defaultData = &testPaymentData{
	ChannelID:    42,
	ChannelNonce: 3,
	Expiration:   time.Now().Add(time.Hour),
	FullAmount:   12345,
	NewAmount:    12345,
	PrevAmount:   12300,
	State:        Open,
}

func copyTestData(orig *testPaymentData) (cpy *testPaymentData) {
	bytes, err := json.Marshal(orig)
	if err != nil {
		panic(fmt.Errorf("Cannot copy test data: %v", err))
	}

	cpy = &testPaymentData{}
	err = json.Unmarshal(bytes, cpy)
	if err != nil {
		panic(fmt.Errorf("Cannot copy test data: %v", err))
	}

	return cpy
}

type D *testPaymentData

func patchDefaultData(patch func(d D)) (cpy *testPaymentData) {
	cpy = copyTestData(defaultData)
	patch(cpy)
	return cpy
}

func getTestPayment(data *testPaymentData) *escrowPaymentType {
	signature := data.Signature
	if signature == nil {
		signature = getSignature(&processorMock.escrowContractAddress, data.ChannelID, data.ChannelNonce, data.NewAmount)
	}
	return &escrowPaymentType{
		grpcContext: &GrpcStreamContext{MD: getEscrowMetadata(data.ChannelID, data.ChannelNonce, data.NewAmount)},
		channelKey:  newPaymentChannelKey(data.ChannelID, data.ChannelNonce),
		amount:      big.NewInt(data.NewAmount),
		signature:   signature,
		channel: &PaymentChannelData{
			State:            data.State,
			Sender:           testPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       data.Expiration,
			AuthorizedAmount: big.NewInt(data.PrevAmount),
			Signature:        nil,
		},
	}
}

func getTestContext(data *testPaymentData) *GrpcStreamContext {
	storageMock.Put(
		newPaymentChannelKey(data.ChannelID, data.ChannelNonce),
		&PaymentChannelData{
			State:            data.State,
			Sender:           testPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       data.Expiration,
			AuthorizedAmount: big.NewInt(data.PrevAmount),
			Signature:        nil,
		},
	)
	md := getEscrowMetadata(data.ChannelID, data.ChannelNonce, data.NewAmount)
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
		processor: &Processor{escrowContractAddress: HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")},
	}
	payment := escrowPaymentType{
		channelKey: newPaymentChannelKey(1789, 1917),
		amount:     big.NewInt(31415),
		// message hash: 04cc38aa4a27976907ef7382182bc549957dc9d2e21eb73651ad6588d5cd4d8f
		signature: HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, HexToAddress("0xc5fdf4076b8f3a5357c5e395ab970b5b54098fef"), *address)
}

func TestGetPublicKeyFromPayment2(t *testing.T) {
	handler := escrowPaymentHandler{
		processor: &Processor{escrowContractAddress: HexToAddress("0x39ee715b50e78a920120c1ded58b1a47f571ab75")},
	}
	payment := escrowPaymentType{
		channelKey: newPaymentChannelKey(1789, 1917),
		amount:     big.NewInt(31415),
		signature:  HexToBytes("0xde4e998341307b036e460b1cc1593ddefe2e9ea261bd6c3d75967b29b2c3d0a24969b4a32b099ae2eded90bbc213ad0a159a66af6d55be7e04f724ffa52ce3cc1b"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, HexToAddress("0x592E3C0f3B038A0D673F19a18a773F993d4b2610"), *address)
}

func TestPaymentChannelToJSON(t *testing.T) {
	channel := PaymentChannelData{
		State:            Open,
		Sender:           testPublicKey,
		FullAmount:       big.NewInt(12345),
		Expiration:       time.Now().Add(time.Hour),
		AuthorizedAmount: big.NewInt(12300),
		Signature:        HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	bytes, err := json.Marshal(channel)
	assert.Nil(t, err)

	channelCopy := &PaymentChannelData{}
	err = json.Unmarshal(bytes, channelCopy)
	assert.Nil(t, err)
}

func TestGetPayment(t *testing.T) {
	data := &testPaymentData{
		ChannelID:    42,
		ChannelNonce: 3,
		Expiration:   time.Now().Add(time.Hour),
		FullAmount:   12345,
		NewAmount:    12345,
		PrevAmount:   12300,
		State:        Open,
	}
	context := getTestContext(data)
	defer clearTestContext()

	payment, err := paymentHandler.Payment(context)

	assert.Nil(t, err)
	expected := getTestPayment(data)
	actual := payment.(*escrowPaymentType)
	assert.Equal(t, toJSON(expected.grpcContext), toJSON(actual.grpcContext))
	assert.Equal(t, toJSON(expected.channelKey), toJSON(actual.channelKey))
	assert.Equal(t, expected.amount, actual.amount)
	assert.Equal(t, expected.signature, actual.signature)
	assert.Equal(t, toJSON(expected.channel), toJSON(actual.channel))
}

func TestGetPaymentNoChannelId(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.ChannelID = 0
	}))
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
}

func TestGetPaymentNoChannelNonce(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.ChannelNonce = 0
	}))
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-nonce\""), err)
}

func TestGetPaymentNoChannelAmount(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.NewAmount = 0
	}))
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-amount\""), err)
}

func TestGetPaymentNoChannel(t *testing.T) {
	context := getTestContext(defaultData)
	storageMock.Clear()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "payment channel \"{ID: 42, Nonce: 3}\" not found"), err)
}

func TestValidatePayment(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		ChannelID:    42,
		ChannelNonce: 3,
		Expiration:   time.Now().Add(time.Hour),
		FullAmount:   12345,
		NewAmount:    12345,
		PrevAmount:   12300,
		State:        Open,
	})

	err := paymentHandler.Validate(payment)

	assert.Nil(t, err)
}

func TestValidatePaymentChannelIsNotOpen(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.State = Closed
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel \"{ID: 42, Nonce: 3}\" is not opened"), err)
}

func TestValidatePaymentIncorrectSignature(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = HexToBytes("0x0000")
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
}

func TestValidatePaymentIncorrectSigner(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01")
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment is not signed by channel sender"), err)
}

func TestValidatePaymentExpiredChannel(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Expiration = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel is expired since \"2009-11-10 23:00:00 +0000 UTC\""), err)
}

func TestValidatePaymentAmountIsTooBig(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.NewAmount = 12346
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: 12345, payment amount: 12346"), err)
}

func TestValidatePaymentIncorrectIncome(t *testing.T) {
	payment := getTestPayment(defaultData)
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
	data := patchDefaultData(func(d D) {
		d.ChannelID = 43
		d.ChannelNonce = 4
		d.FullAmount = 12346
		d.NewAmount = 12345
	})
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
	data := patchDefaultData(func(d D) {
		d.ChannelID = 43
		d.ChannelNonce = 4
		d.FullAmount = 12346
		d.NewAmount = 12345
	})
	payment := getTestPayment(data)

	err := paymentHandler.Complete(payment)

	assert.Equal(t, status.New(codes.Internal, "unable to store new payment channel state"), err)
}
