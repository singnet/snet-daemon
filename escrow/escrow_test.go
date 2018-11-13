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

type paymentChannelServiceMock struct {
	lockingPaymentChannelService

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

func (service *paymentChannelServiceMock) StartPaymentTransaction(payment *Payment) (PaymentTransaction, error) {
	if service.err != nil {
		return nil, service.err
	}

	return &paymentTransactionMock{
		channel: service.data,
		err:     service.err,
	}, nil
}

type paymentTransactionMock struct {
	channel *PaymentChannelData
	err     error
}

func (transaction *paymentTransactionMock) Channel() *PaymentChannelData {
	return transaction.channel
}

func (transaction *paymentTransactionMock) Commit() error {
	return transaction.err
}

func (transaction *paymentTransactionMock) Rollback() error {
	return transaction.err
}

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

	var paymentChannelService = &lockingPaymentChannelService{
		storage:          storageMock,
		blockchainReader: NewBlockchainChannelReaderMock(),
		locker:           &lockerMock{},
		validator:        ChannelPaymentValidatorMock(),
	}
	var paymentHandler = &paymentChannelPaymentHandler{
		service:            paymentChannelService,
		mpeContractAddress: func() common.Address { return testEscrowContractAddress },
		incomeValidator:    incomeValidatorMock,
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
