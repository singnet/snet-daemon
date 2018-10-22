package escrow

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
	"strconv"
	"testing"
	"time"
)

var testPrivateKey = generatePrivateKey()
var testPublicKey = crypto.PubkeyToAddress(testPrivateKey.PublicKey)
var recipientPrivateKey = generatePrivateKey()
var recipientPublicKey = crypto.PubkeyToAddress(recipientPrivateKey.PublicKey)

func generatePrivateKey() (privateKey *ecdsa.PrivateKey) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Cannot generate private key for test: %v", err))
	}
	return
}

type storageMockType struct {
	delegate PaymentChannelStorage
	errors   map[string]bool
}

var storageMock = storageMockType{
	delegate: NewPaymentChannelStorage(NewMemStorage()),
	errors:   make(map[string]bool),
}

func (storage *storageMockType) Put(key *PaymentChannelKey, channel *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, channel)
}

func getMemoryStorageKey(key *PaymentChannelKey) string {
	return key.String()
}

func (storage *storageMockType) Get(_key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	key := getMemoryStorageKey(_key)
	if storage.errors[key] {
		return nil, false, errors.New("storage error")
	}
	return storage.delegate.Get(_key)
}

func (storage *storageMockType) CompareAndSwap(_key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	key := getMemoryStorageKey(_key)
	if storage.errors[key] {
		return false, errors.New("storage error")
	}
	return storage.delegate.CompareAndSwap(_key, prevState, newState)
}

func (storage *storageMockType) Clear() {
	storage.delegate = NewPaymentChannelStorage(NewMemStorage())
	storage.errors = make(map[string]bool)
}

func (storage *storageMockType) SetError(key *PaymentChannelKey, err bool) {
	storage.errors[getMemoryStorageKey(key)] = err
}

type incomeValidatorMockType struct {
	err *status.Status
}

var incomeValidatorMock = incomeValidatorMockType{}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeData) (err *status.Status) {
	return incomeValidator.err
}

var testEscrowContractAddress = blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")

var paymentHandler = escrowPaymentHandler{
	escrowContractAddress: testEscrowContractAddress,
	storage:               &storageMock,
	incomeValidator:       &incomeValidatorMock,
}

func getSignature(contractAddress *common.Address, channelID, channelNonce, amount int64, privateKey *ecdsa.PrivateKey) (signature []byte) {
	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(
			testEscrowContractAddress.Bytes(),
			intToUint256(channelID),
			intToUint256(channelNonce),
			intToUint256(amount),
		),
	)

	signature, err := crypto.Sign(hash, privateKey)
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
	md.Set(PaymentChannelSignatureHeader, string(getSignature(&testEscrowContractAddress, channelID, channelNonce, amount, testPrivateKey)))
	return md
}

type testPaymentData struct {
	ChannelID, ChannelNonce, FullAmount, PrevAmount, NewAmount, GroupId int64
	State                                                               PaymentChannelState
	Expiration                                                          time.Time
	Signature                                                           []byte
}

func newPaymentChannelKey(ID, nonce int64) *PaymentChannelKey {
	return &PaymentChannelKey{ID: big.NewInt(ID)}
}

var defaultData = &testPaymentData{
	ChannelID:    42,
	ChannelNonce: 3,
	Expiration:   time.Now().Add(time.Hour),
	FullAmount:   12345,
	NewAmount:    12345,
	PrevAmount:   12300,
	State:        Open,
	GroupId:      1,
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
		signature = getSignature(&testEscrowContractAddress, data.ChannelID, data.ChannelNonce, data.NewAmount, testPrivateKey)
	}
	return &escrowPaymentType{
		grpcContext:  &handler.GrpcStreamContext{MD: getEscrowMetadata(data.ChannelID, data.ChannelNonce, data.NewAmount)},
		channelID:    big.NewInt(data.ChannelID),
		channelNonce: big.NewInt(data.ChannelNonce),
		amount:       big.NewInt(data.NewAmount),
		signature:    signature,
		channel: &PaymentChannelData{
			Nonce:            big.NewInt(data.ChannelNonce),
			State:            data.State,
			Sender:           testPublicKey,
			Recipient:        recipientPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       data.Expiration,
			AuthorizedAmount: big.NewInt(data.PrevAmount),
			Signature:        nil,
			GroupId:          big.NewInt(data.GroupId),
		},
	}
}

func getTestContext(data *testPaymentData) *handler.GrpcStreamContext {
	storageMock.Put(
		newPaymentChannelKey(data.ChannelID, data.ChannelNonce),
		&PaymentChannelData{
			Nonce:            big.NewInt(data.ChannelNonce),
			State:            data.State,
			Sender:           testPublicKey,
			Recipient:        recipientPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       data.Expiration,
			AuthorizedAmount: big.NewInt(data.PrevAmount),
			Signature:        nil,
			GroupId:          big.NewInt(data.GroupId),
		},
	)
	md := getEscrowMetadata(data.ChannelID, data.ChannelNonce, data.NewAmount)
	return &handler.GrpcStreamContext{
		MD: md,
	}
}

func clearTestContext() {
	storageMock.Clear()
}

func toJSON(data interface{}) string {
	return bytesErrorTupleToString(json.Marshal(data))
}

func bytesErrorTupleToString(data []byte, err error) string {
	if err != nil {
		panic(fmt.Sprintf("Unexpected error: %v", err))
	}
	return string(data)
}

func TestGetPublicKeyFromPayment(t *testing.T) {
	handler := escrowPaymentHandler{
		escrowContractAddress: testEscrowContractAddress,
	}
	payment := escrowPaymentType{
		channelID:    big.NewInt(1789),
		channelNonce: big.NewInt(1917),
		amount:       big.NewInt(31415),
		// message hash: 04cc38aa4a27976907ef7382182bc549957dc9d2e21eb73651ad6588d5cd4d8f
		signature: blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, blockchain.HexToAddress("0xc5fdf4076b8f3a5357c5e395ab970b5b54098fef"), *address)
}

func TestGetPublicKeyFromPayment2(t *testing.T) {
	handler := escrowPaymentHandler{
		escrowContractAddress: blockchain.HexToAddress("0x39ee715b50e78a920120c1ded58b1a47f571ab75"),
	}
	payment := escrowPaymentType{
		channelID:    big.NewInt(1789),
		channelNonce: big.NewInt(1917),
		amount:       big.NewInt(31415),
		signature:    blockchain.HexToBytes("0xde4e998341307b036e460b1cc1593ddefe2e9ea261bd6c3d75967b29b2c3d0a24969b4a32b099ae2eded90bbc213ad0a159a66af6d55be7e04f724ffa52ce3cc1b"),
	}

	address, err := handler.getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, blockchain.HexToAddress("0x592E3C0f3B038A0D673F19a18a773F993d4b2610"), *address)
}

func TestPaymentChannelToJSON(t *testing.T) {
	channel := PaymentChannelData{
		Nonce:            big.NewInt(3),
		State:            Open,
		Sender:           testPublicKey,
		Recipient:        recipientPublicKey,
		FullAmount:       big.NewInt(12345),
		Expiration:       time.Now().Add(time.Hour),
		AuthorizedAmount: big.NewInt(12300),
		Signature:        blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
		GroupId:          big.NewInt(1),
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
	assert.Equal(t, toJSON(expected.channelID), toJSON(actual.channelID))
	assert.Equal(t, toJSON(expected.channelNonce), toJSON(actual.channelNonce))
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

func TestGetPaymentStorageError(t *testing.T) {
	context := getTestContext(defaultData)
	storageMock.SetError(newPaymentChannelKey(defaultData.ChannelID, defaultData.ChannelNonce), true)
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.Internal, "payment channel storage error"), err)
}

func TestGetPaymentNoChannel(t *testing.T) {
	context := getTestContext(defaultData)
	storageMock.Clear()
	defer clearTestContext()

	_, err := paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "payment channel \"{ID: 42}\" not found"), err)
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

func TestValidatePaymentChannelNonce(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.ChannelNonce = 3
	}))
	payment.channelNonce = big.NewInt(2)

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "incorrect payment channel nonce, latest: 3, sent: 2"), err)
}

func TestValidatePaymentIncorrectSignatureLength(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0x0000")
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
}

func TestValidatePaymentIncorrectSignatureChecksum(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef21")
	}))

	err := paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
}

func TestValidatePaymentIncorrectSigner(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01")
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
		escrowContractAddress: testEscrowContractAddress,
		storage:               &storageMock,
		incomeValidator:       &incomeValidatorMockType{err: incomeErr},
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
	defer clearTestContext()
	payment := getTestPayment(data)

	err := paymentHandler.Complete(payment)
	channelState, ok, e := storageMock.Get(newPaymentChannelKey(43, 4))

	assert.Nil(t, err)
	assert.Nil(t, e)
	assert.True(t, ok)
	assert.Equal(t, toJSON(&PaymentChannelData{
		Nonce:            big.NewInt(4),
		State:            Open,
		Sender:           testPublicKey,
		Recipient:        recipientPublicKey,
		FullAmount:       big.NewInt(12346),
		Expiration:       payment.channel.Expiration,
		AuthorizedAmount: big.NewInt(12345),
		Signature:        payment.signature,
		GroupId:          big.NewInt(1),
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
	storageMock.SetError(newPaymentChannelKey(43, 4), true)
	defer clearTestContext()

	err := paymentHandler.Complete(payment)

	assert.Equal(t, status.New(codes.Internal, "unable to store new payment channel state"), err)
}

func TestCompletePaymentConcurrentUpdate(t *testing.T) {
	data := patchDefaultData(func(d D) {
		d.ChannelID = 43
		d.ChannelNonce = 4
		d.FullAmount = 12346
		d.NewAmount = 12345
	})
	clearTestContext()
	payment := getTestPayment(data)

	err := paymentHandler.Complete(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "state of payment channel was concurrently updated, channel id: 43"), err)
}
