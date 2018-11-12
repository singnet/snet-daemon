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
	paymentChannelService     *lockingPaymentChannelService
	paymentHandler            *paymentChannelPaymentHandler
	defaultData               *testPaymentData
	configMock                *viper.Viper
}

type blockchainMockType struct {
	escrowContractAddress common.Address
	currentBlock          int64
	err                   error
}

func (mock *blockchainMockType) EscrowContractAddress() common.Address {
	return mock.escrowContractAddress
}

func (mock *blockchainMockType) CurrentBlock() (currentBlock *big.Int, err error) {
	if mock.err != nil {
		return nil, mock.err
	}
	return big.NewInt(mock.currentBlock), nil
}

func (mock *blockchainMockType) MultiPartyEscrowChannel(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
	return nil, false, nil
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
	configMock.Set(config.PaymentExpirationThresholdBlocksKey, 0)

	var blockchainMock = &blockchainMockType{
		escrowContractAddress: testEscrowContractAddress,
		currentBlock:          99,
	}

	var paymentChannelService = &lockingPaymentChannelService{
		config:     configMock,
		storage:    storageMock,
		blockchain: blockchainMock,
		locker:     &lockerMock{},
	}
	var paymentHandler = &paymentChannelPaymentHandler{
		service:         paymentChannelService,
		blockchain:      blockchainMock,
		incomeValidator: incomeValidatorMock,
	}
	var defaultData = &testPaymentData{
		ChannelID:           42,
		ChannelNonce:        3,
		PaymentChannelNonce: 3,
		Expiration:          100,
		FullAmount:          12345,
		NewAmount:           12345,
		PrevAmount:          12300,
		State:               Open,
		GroupId:             1,
		Signature:           getPaymentSignature(&testEscrowContractAddress, 42, 3, 12345, testPrivateKey),
	}

	return &escrowTestType{
		testPrivateKey:            testPrivateKey,
		testPublicKey:             testPublicKey,
		recipientPublicKey:        recipientPublicKey,
		storageMock:               storageMock,
		testEscrowContractAddress: testEscrowContractAddress,
		paymentChannelService:     paymentChannelService,
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

func (storage *storageMockType) Get(_key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	if storage.err != nil {
		return nil, false, storage.err
	}
	return storage.delegate.Get(_key)
}

func (storage *storageMockType) PutIfAbsent(key *PaymentChannelKey, channel *PaymentChannelData) (ok bool, err error) {
	if storage.err != nil {
		return false, storage.err
	}
	return storage.delegate.PutIfAbsent(key, channel)
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

func getEscrowMetadata(channelID, channelNonce, amount int64, signature []byte) metadata.MD {
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
	md.Set(PaymentChannelSignatureHeader, string(signature))
	return md
}

type testPaymentData struct {
	ChannelID, ChannelNonce, PaymentChannelNonce, FullAmount, PrevAmount, NewAmount, GroupId int64
	State                                                                                    PaymentChannelState
	Expiration                                                                               int64
	Signature                                                                                []byte
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

func getTestPayment(data *testPaymentData) *paymentTransaction {
	signature := data.Signature
	if signature == nil {
		signature = getPaymentSignature(&escrowTest.testEscrowContractAddress, data.ChannelID, data.PaymentChannelNonce, data.NewAmount, escrowTest.testPrivateKey)
	}
	return &paymentTransaction{
		payment: Payment{
			MpeContractAddress: escrowTest.testEscrowContractAddress,
			ChannelID:          big.NewInt(data.ChannelID),
			ChannelNonce:       big.NewInt(data.PaymentChannelNonce),
			Amount:             big.NewInt(data.NewAmount),
			Signature:          signature,
		},
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
		service: escrowTest.paymentChannelService,
		lock:    &lockMock{},
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
	md := getEscrowMetadata(data.ChannelID, data.PaymentChannelNonce, data.NewAmount, data.Signature)
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
	payment := Payment{
		MpeContractAddress: escrowTest.testEscrowContractAddress,
		ChannelID:          big.NewInt(1789),
		ChannelNonce:       big.NewInt(1917),
		Amount:             big.NewInt(31415),
		// message hash: 04cc38aa4a27976907ef7382182bc549957dc9d2e21eb73651ad6588d5cd4d8f
		Signature: blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	address, err := getSignerAddressFromPayment(&payment)

	assert.Nil(t, err)
	assert.Equal(t, blockchain.HexToAddress("0xc5fdf4076b8f3a5357c5e395ab970b5b54098fef"), *address)
}

func TestGetPublicKeyFromPayment2(t *testing.T) {
	payment := Payment{
		MpeContractAddress: blockchain.HexToAddress("0x39ee715b50e78a920120c1ded58b1a47f571ab75"),
		ChannelID:          big.NewInt(1789),
		ChannelNonce:       big.NewInt(1917),
		Amount:             big.NewInt(31415),
		Signature:          blockchain.HexToBytes("0xde4e998341307b036e460b1cc1593ddefe2e9ea261bd6c3d75967b29b2c3d0a24969b4a32b099ae2eded90bbc213ad0a159a66af6d55be7e04f724ffa52ce3cc1b"),
	}

	address, err := getSignerAddressFromPayment(&payment)

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
		ChannelID:           42,
		ChannelNonce:        3,
		PaymentChannelNonce: 3,
		Expiration:          100,
		FullAmount:          12345,
		NewAmount:           12345,
		PrevAmount:          12300,
		State:               Open,
		Signature:           getPaymentSignature(&escrowTest.testEscrowContractAddress, 42, 3, 12345, escrowTest.testPrivateKey),
	}
	context := getTestContext(data)
	defer clearTestContext()

	payment, err := escrowTest.paymentHandler.Payment(context)

	assert.Nil(t, err)
	expected := getTestPayment(data)
	actual := payment.(*paymentTransaction)
	assert.Equal(t, toJSON(expected.payment.ChannelID), toJSON(actual.payment.ChannelID))
	assert.Equal(t, toJSON(expected.payment.ChannelNonce), toJSON(actual.payment.ChannelNonce))
	assert.Equal(t, expected.payment.Amount, actual.payment.Amount)
	assert.Equal(t, expected.payment.Signature, actual.payment.Signature)
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
		d.PaymentChannelNonce = 0
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

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel \"{ID: 42}\" not found"), err)
}

func TestValidatePaymentChannelNonce(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.ChannelNonce = 3
		d.PaymentChannelNonce = 2
		d.Signature = getPaymentSignature(
			&escrowTest.testEscrowContractAddress,
			escrowTest.defaultData.ChannelID,
			2,
			escrowTest.defaultData.NewAmount,
			escrowTest.testPrivateKey)
	}))

	payment, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.Unauthenticated, "incorrect payment channel nonce, latest: 3, sent: 2"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentIncorrectSignatureLength(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0x0000")
	}))

	payment, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentIncorrectSignatureChecksum(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef21")
	}))

	payment, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment signature is not valid"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentIncorrectSigner(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.Signature = blockchain.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01")
	}))

	payment, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment is not signed by channel sender"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentChannelCannotGetCurrentBlock(t *testing.T) {
	service := escrowTest.paymentChannelService
	service.blockchain = &blockchainMockType{
		escrowContractAddress: escrowTest.testEscrowContractAddress,
		err: errors.New("blockchain error"),
	}
	handler := escrowTest.paymentHandler
	handler.service = service
	context := getTestContext(patchDefaultData(func(d D) {
		d.Expiration = 99
	}))

	payment, err := handler.Payment(context)

	assert.Equal(t, status.New(codes.Internal, "cannot determine current block"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentExpiredChannel(t *testing.T) {
	service := escrowTest.paymentChannelService
	service.blockchain = &blockchainMockType{
		escrowContractAddress: escrowTest.testEscrowContractAddress,
		currentBlock:          99,
	}
	handler := escrowTest.paymentHandler
	handler.service = service
	context := getTestContext(patchDefaultData(func(d D) {
		d.Expiration = 99
	}))

	payment, err := handler.Payment(context)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 99, expiration threshold: 0"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentChannelExpirationThreshold(t *testing.T) {
	service := escrowTest.paymentChannelService
	service.config = viper.New()
	service.config.Set(config.PaymentExpirationThresholdBlocksKey, 1)
	service.blockchain = &blockchainMockType{
		escrowContractAddress: escrowTest.testEscrowContractAddress,
		currentBlock:          98,
	}
	handler := escrowTest.paymentHandler
	handler.service = service
	context := getTestContext(patchDefaultData(func(d D) {
		d.Expiration = 99
	}))

	payment, err := handler.Payment(context)

	assert.Equal(t, status.New(codes.Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 98, expiration threshold: 1"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentAmountIsTooBig(t *testing.T) {
	context := getTestContext(patchDefaultData(func(d D) {
		d.NewAmount = 12346
		d.Signature = getPaymentSignature(
			&escrowTest.testEscrowContractAddress,
			escrowTest.defaultData.ChannelID,
			escrowTest.defaultData.PaymentChannelNonce,
			12346,
			escrowTest.testPrivateKey)
	}))

	payment, err := escrowTest.paymentHandler.Payment(context)

	assert.Equal(t, status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: 12345, payment amount: 12346"), err)
	assert.Nil(t, payment)
}

func TestValidatePaymentIncorrectIncome(t *testing.T) {
	context := getTestContext(escrowTest.defaultData)
	incomeErr := status.New(codes.Unauthenticated, "incorrect payment income: \"45\", expected \"46\"")
	blockchain := &blockchainMockType{escrowContractAddress: escrowTest.testEscrowContractAddress}
	paymentHandler := paymentChannelPaymentHandler{
		service: &lockingPaymentChannelService{
			config:     escrowTest.configMock,
			storage:    escrowTest.storageMock,
			blockchain: blockchain,
			locker:     &lockerMock{},
		},
		incomeValidator: &incomeValidatorMockType{err: incomeErr},
		blockchain:      blockchain,
	}

	payment, err := paymentHandler.Payment(context)

	assert.Equal(t, incomeErr, err)
	assert.Nil(t, payment)
}

func TestCompletePayment(t *testing.T) {
	data := patchDefaultData(func(d D) {
		d.ChannelID = 43
		d.ChannelNonce = 4
		d.PaymentChannelNonce = 4
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
		Signature:        payment.payment.Signature,
		GroupId:          big.NewInt(1),
	}), toJSON(channelState))
}

func TestCompletePaymentCannotUpdateChannel(t *testing.T) {
	data := patchDefaultData(func(d D) {
		d.ChannelID = 43
		d.ChannelNonce = 4
		d.PaymentChannelNonce = 4
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
		d.PaymentChannelNonce = 4
		d.FullAmount = 12346
		d.NewAmount = 12345
	})
	clearTestContext()
	payment := getTestPayment(data)

	err := escrowTest.paymentHandler.Complete(payment)

	assert.Equal(t, status.New(codes.Unauthenticated, "state of payment channel was concurrently updated, channel id: 43"), err)
}
