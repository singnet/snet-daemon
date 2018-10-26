package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
)

type escrowTestType struct {
	testPrivateKey            *ecdsa.PrivateKey
	testPublicKey             common.Address
	recipientPublicKey        common.Address
	storageMock               *storageMockType
	testEscrowContractAddress common.Address
	paymentHandler            *escrowPaymentHandler
	defaultData               *testPaymentData
	configMock                *viper.Viper
}

type blockchainMockType struct {
	escrowContractAddress common.Address
	currentBlock          int64
}

func (mock *blockchainMockType) EscrowContractAddress() common.Address {
	return mock.escrowContractAddress
}

func (mock *blockchainMockType) CurrentBlock() (currentBlock *big.Int, err error) {
	return big.NewInt(mock.currentBlock), nil
}

var escrowTest = func() *escrowTestType {

	var testPrivateKey = generatePrivateKey()
	var testPublicKey = crypto.PubkeyToAddress(testPrivateKey.PublicKey)
	var recipientPrivateKey = generatePrivateKey()
	var recipientPublicKey = crypto.PubkeyToAddress(recipientPrivateKey.PublicKey)
	var storageMock = &storageMockType{
		delegate: NewPaymentChannelStorage(NewMemStorage()),
		err:      nil,
	}
	var incomeValidatorMock = &incomeValidatorMockType{}

	var testEscrowContractAddress = blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")

	var configMock = viper.New()
	configMock.Set(config.PaymentExpirationTresholdBlocksKey, 0)

	var blockchainMock = &blockchainMockType{
		escrowContractAddress: testEscrowContractAddress,
		currentBlock:          99,
	}

	var paymentHandler = &escrowPaymentHandler{
		config:          configMock,
		storage:         storageMock,
		incomeValidator: incomeValidatorMock,
		blockchain:      blockchainMock,
	}
	var defaultData = &testPaymentData{
		ChannelID:    42,
		ChannelNonce: 3,
		Expiration:   100,
		FullAmount:   12345,
		NewAmount:    12345,
		PrevAmount:   12300,
		State:        Open,
		GroupId:      1,
	}

	return &escrowTestType{
		testPrivateKey:            testPrivateKey,
		testPublicKey:             testPublicKey,
		recipientPublicKey:        recipientPublicKey,
		storageMock:               storageMock,
		testEscrowContractAddress: testEscrowContractAddress,
		paymentHandler:            paymentHandler,
		defaultData:               defaultData,
		configMock:                configMock,
	}
}()

func generatePrivateKey() (privateKey *ecdsa.PrivateKey) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Cannot generate private key for test: %v", err))
	}
	return
}

type storageMockType struct {
	delegate PaymentChannelStorage
	err      error
}

func (storage *storageMockType) Put(key *PaymentChannelKey, channel *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, channel)
}

func getMemoryStorageKey(key *PaymentChannelKey) string {
	return key.String()
}

func (storage *storageMockType) Get(_key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	if storage.err != nil {
		return nil, false, storage.err
	}
	return storage.delegate.Get(_key)
}

func (storage *storageMockType) CompareAndSwap(_key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	if storage.err != nil {
		return false, storage.err
	}
	return storage.delegate.CompareAndSwap(_key, prevState, newState)
}

func (storage *storageMockType) Clear() {
	storage.delegate = NewPaymentChannelStorage(NewMemStorage())
	storage.err = nil
}

func (storage *storageMockType) SetError(err error) {
	storage.err = err
}

type incomeValidatorMockType struct {
	err *status.Status
}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeData) (err *status.Status) {
	return incomeValidator.err
}

func getPaymentSignature(contractAddress *common.Address, channelID, channelNonce, amount int64, privateKey *ecdsa.PrivateKey) (signature []byte) {
	message := bytes.Join([][]byte{
		contractAddress.Bytes(),
		intToUint256(channelID),
		intToUint256(channelNonce),
		intToUint256(amount),
	}, nil)

	return getSignature(message, privateKey)
}

func getSignature(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte) {
	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
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
	md.Set(PaymentChannelSignatureHeader, string(getPaymentSignature(&escrowTest.testEscrowContractAddress, channelID, channelNonce, amount, escrowTest.testPrivateKey)))
	return md
}

type testPaymentData struct {
	ChannelID, ChannelNonce, FullAmount, PrevAmount, NewAmount, GroupId int64
	State                                                               PaymentChannelState
	Expiration                                                          int64
	Signature                                                           []byte
}

func newPaymentChannelKey(ID, nonce int64) *PaymentChannelKey {
	return &PaymentChannelKey{ID: big.NewInt(ID)}
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
	cpy = copyTestData(escrowTest.defaultData)
	patch(cpy)
	return cpy
}

func getTestPayment(data *testPaymentData) *escrowPaymentType {
	signature := data.Signature
	if signature == nil {
		signature = getPaymentSignature(&escrowTest.testEscrowContractAddress, data.ChannelID, data.ChannelNonce, data.NewAmount, escrowTest.testPrivateKey)
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
			Sender:           escrowTest.testPublicKey,
			Recipient:        escrowTest.recipientPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       big.NewInt(data.Expiration),
			AuthorizedAmount: big.NewInt(data.PrevAmount),
			Signature:        nil,
			GroupId:          big.NewInt(data.GroupId),
		},
	}
}

func getTestContext(data *testPaymentData) *handler.GrpcStreamContext {
	escrowTest.storageMock.Put(
		newPaymentChannelKey(data.ChannelID, data.ChannelNonce),
		&PaymentChannelData{
			Nonce:            big.NewInt(data.ChannelNonce),
			State:            data.State,
			Sender:           escrowTest.testPublicKey,
			Recipient:        escrowTest.recipientPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       big.NewInt(data.Expiration),
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
	escrowTest.storageMock.Clear()
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
		blockchain: &blockchainMockType{escrowContractAddress: escrowTest.testEscrowContractAddress},
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
		blockchain: &blockchainMockType{escrowContractAddress: blockchain.HexToAddress("0x39ee715b50e78a920120c1ded58b1a47f571ab75")},
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
		Sender:           escrowTest.testPublicKey,
		Recipient:        escrowTest.recipientPublicKey,
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
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
		Expiration:   100,
		FullAmount:   12345,
		NewAmount:    12345,
		PrevAmount:   12300,
		State:        Open,
	}
	context := getTestContext(data)
	defer clearTestContext()

	payment, err := escrowTest.paymentHandler.Payment(context)

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

	_, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
}

func TestGetPaymentNoChannelNonce(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.ChannelNonce = 0
	}))
	defer clearTestContext()

	_, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-nonce\""), err)
}

func TestGetPaymentNoChannelAmount(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.NewAmount = 0
	}))
	defer clearTestContext()

	_, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "missing \"snet-payment-channel-amount\""), err)
}

func TestGetPaymentStorageError(t *testing.T) {
	context := getTestContext(escrowTest.defaultData)
	escrowTest.storageMock.SetError(errors.New("storage error"))
	defer clearTestContext()

	_, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.Internal, "payment channel storage error"), err)
}

func TestGetPaymentNoChannel(t *testing.T) {
	context := getTestContext(escrowTest.defaultData)
	escrowTest.storageMock.Clear()
	defer clearTestContext()

	_, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.InvalidArgument, "payment channel \"{ID: 42}\" not found"), err)
}

func TestValidatePayment(t *testing.T) {
	payment := getTestPayment(&testPaymentData{
		ChannelID:    42,
		ChannelNonce: 3,
		Expiration:   100,
		FullAmount:   12345,
		NewAmount:    12345,
		PrevAmount:   12300,
		State:        Open,
	})

	err := escrowTest.paymentHandler.Validate(payment)

	assert.Nil(t, err, "Unexpected error: %v", err.Message())
}

func TestValidatePaymentChannelNonce(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.ChannelNonce = 3
	}))
	payment.channelNonce = big.NewInt(2)

	err := escrowTest.paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "incorrect payment channel nonce, latest: 3, sent: 2"), err)
}

func TestValidatePaymentIncorrectSignatureLength(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0x0000")
	}))

	err := escrowTest.paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
}

func TestValidatePaymentIncorrectSignatureChecksum(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef21")
	}))

	err := escrowTest.paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
}

func TestValidatePaymentIncorrectSigner(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01")
	}))

	err := escrowTest.paymentHandler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment is not signed by channel sender"), err)
}

func TestValidatePaymentExpiredChannel(t *testing.T) {
	handler := escrowTest.paymentHandler
	handler.blockchain = &blockchainMockType{
		escrowContractAddress: escrowTest.testEscrowContractAddress,
		currentBlock:          99,
	}
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Expiration = 99
	}))

	err := handler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 99, expiration treshold: 0"), err)
}

func TestValidatePaymentChannelExpirationTreshold(t *testing.T) {
	handler := escrowTest.paymentHandler
	handler.config = viper.New()
	handler.config.Set(config.PaymentExpirationTresholdBlocksKey, 1)
	handler.blockchain = &blockchainMockType{
		escrowContractAddress: escrowTest.testEscrowContractAddress,
		currentBlock:          98,
	}
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.Expiration = 99
	}))

	err := handler.Validate(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 98, expiration treshold: 1"), err)
}

func TestValidatePaymentAmountIsTooBig(t *testing.T) {
	payment := getTestPayment(patchDefaultData(func(d D) {
		d.NewAmount = 12346
	}))

	err := escrowTest.paymentHandler.Validate(payment)

	assert.Equal(t, status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: 12345, payment amount: 12346"), err)
}

func TestValidatePaymentIncorrectIncome(t *testing.T) {
	payment := getTestPayment(escrowTest.defaultData)
	incomeErr := status.New(codes.Unauthenticated, "incorrect payment income: \"45\", expected \"46\"")
	paymentHandler := escrowPaymentHandler{
		config:          escrowTest.configMock,
		storage:         escrowTest.storageMock,
		incomeValidator: &incomeValidatorMockType{err: incomeErr},
		blockchain:      &blockchainMockType{escrowContractAddress: escrowTest.testEscrowContractAddress},
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

	err := escrowTest.paymentHandler.Complete(payment)
	channelState, ok, e := escrowTest.storageMock.Get(newPaymentChannelKey(43, 4))

	assert.Nil(t, err)
	assert.Nil(t, e)
	assert.True(t, ok)
	assert.Equal(t, toJSON(&PaymentChannelData{
		Nonce:            big.NewInt(4),
		State:            Open,
		Sender:           escrowTest.testPublicKey,
		Recipient:        escrowTest.recipientPublicKey,
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
	escrowTest.storageMock.SetError(errors.New("storage error"))
	defer clearTestContext()

	err := escrowTest.paymentHandler.Complete(payment)

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

	err := escrowTest.paymentHandler.Complete(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "state of payment channel was concurrently updated, channel id: 43"), err)
}
