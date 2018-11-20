package escrow

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/singnet/snet-daemon/blockchain"
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

func (p *paymentChannelServiceMock) StartPaymentTransaction(payment *Payment) (PaymentTransaction, error) {
	if p.err != nil {
		return nil, p.err
	}

	return &paymentTransactionMock{
		channel: p.data,
		err:     p.err,
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

type PaymentChannelServiceSuite struct {
	suite.Suite

	senderPrivateKey   *ecdsa.PrivateKey
	senderAddress      common.Address
	recipientAddress   common.Address
	mpeContractAddress common.Address
	memoryStorage      *memoryStorage
	storage            *PaymentChannelStorage
	paymentStorage     *PaymentStorage

	service PaymentChannelService
}

func (suite *PaymentChannelServiceSuite) SetupSuite() {
	suite.senderPrivateKey = GenerateTestPrivateKey()
	suite.senderAddress = crypto.PubkeyToAddress(suite.senderPrivateKey.PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.mpeContractAddress = blockchain.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")
	suite.memoryStorage = NewMemStorage()
	suite.storage = NewPaymentChannelStorage(suite.memoryStorage)
	suite.paymentStorage = NewPaymentStorage(suite.memoryStorage)

	err := suite.storage.Put(suite.channelKey(), suite.channel())
	if err != nil {
		panic(fmt.Errorf("Cannot put value into test storage: %v", err))
	}

	suite.service = NewPaymentChannelService(
		suite.storage,
		suite.paymentStorage,
		&BlockchainChannelReader{
			replicaGroupID: func() ([32]byte, error) {
				return [32]byte{123}, nil
			},
			readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
				return suite.mpeChannel(), true, nil
			},
		},
		NewEtcdLocker(suite.memoryStorage),
		&ChannelPaymentValidator{
			currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
			paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
		},
	)
}

func (suite *PaymentChannelServiceSuite) SetupTest() {
	suite.memoryStorage.Clear()
}

func TestPaymentChannelServiceSuite(t *testing.T) {
	suite.Run(t, new(PaymentChannelServiceSuite))
}

func (suite *PaymentChannelServiceSuite) mpeChannel() *blockchain.MultiPartyEscrowChannel {
	return &blockchain.MultiPartyEscrowChannel{
		Sender:     suite.senderAddress,
		Recipient:  suite.recipientAddress,
		GroupId:    [32]byte{123},
		Value:      big.NewInt(12345),
		Nonce:      big.NewInt(3),
		Expiration: big.NewInt(100),
	}
}

func (suite *PaymentChannelServiceSuite) payment() *Payment {
	payment := &Payment{
		Amount:       big.NewInt(12300),
		ChannelID:    big.NewInt(42),
		ChannelNonce: big.NewInt(3),
		//MpeContractAddress: suite.mpeContractAddress,
	}
	SignTestPayment(payment, suite.senderPrivateKey)
	return payment
}

func (suite *PaymentChannelServiceSuite) channelKey() *PaymentChannelKey {
	return &PaymentChannelKey{
		ID: big.NewInt(42),
	}
}

func (suite *PaymentChannelServiceSuite) channel() *PaymentChannelData {
	return &PaymentChannelData{
		ChannelID:        big.NewInt(42),
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupID:          [32]byte{123},
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}
}

func (suite *PaymentChannelServiceSuite) channelPlusPayment(payment *Payment) *PaymentChannelData {
	channel := suite.channel()
	channel.Signature = payment.Signature
	channel.AuthorizedAmount = payment.Amount
	return channel
}

func (suite *PaymentChannelServiceSuite) TestPaymentTransaction() {
	payment := suite.payment()

	transaction, errA := suite.service.StartPaymentTransaction(payment)
	errB := transaction.Commit()
	channel, ok, errC := suite.storage.Get(suite.channelKey())

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	assert.Nil(suite.T(), errC, "Unexpected error: %v", errC)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), suite.channelPlusPayment(payment), channel)
}

func (suite *PaymentChannelServiceSuite) TestPaymentParallelTransaction() {
	paymentA := suite.payment()
	paymentA.Amount = big.NewInt(13)
	SignTestPayment(paymentA, suite.senderPrivateKey)
	paymentB := suite.payment()
	paymentB.Amount = big.NewInt(17)
	SignTestPayment(paymentB, suite.senderPrivateKey)

	transactionA, errA := suite.service.StartPaymentTransaction(paymentA)
	transactionB, errB := suite.service.StartPaymentTransaction(paymentB)
	errC := transactionA.Commit()
	channel, ok, errD := suite.storage.Get(suite.channelKey())

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Equal(suite.T(), NewPaymentError(FailedPrecondition, "another transaction on channel: {ID: 42} is in progress"), errB)
	assert.Nil(suite.T(), transactionB)
	assert.Nil(suite.T(), errC, "Unexpected error: %v", errC)
	assert.Nil(suite.T(), errD, "Unexpected error: %v", errD)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), suite.channelPlusPayment(paymentA), channel)
}

func (suite *PaymentChannelServiceSuite) TestPaymentSequentialTransaction() {
	paymentA := suite.payment()
	paymentA.Amount = big.NewInt(13)
	SignTestPayment(paymentA, suite.senderPrivateKey)
	paymentB := suite.payment()
	paymentB.Amount = big.NewInt(17)
	SignTestPayment(paymentB, suite.senderPrivateKey)

	transactionA, errA := suite.service.StartPaymentTransaction(paymentA)
	errAC := transactionA.Commit()
	transactionB, errB := suite.service.StartPaymentTransaction(paymentB)
	errBC := transactionB.Commit()
	channel, ok, errD := suite.storage.Get(suite.channelKey())

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Nil(suite.T(), errAC, "Unexpected error: %v", errAC)
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	assert.Nil(suite.T(), errBC, "Unexpected error: %v", errBC)
	assert.Nil(suite.T(), errD, "Unexpected error: %v", errD)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), suite.channelPlusPayment(paymentB), channel)
}

func (suite *PaymentChannelServiceSuite) TestPaymentSequentialTransactionAfterRollback() {
	paymentA := suite.payment()
	paymentA.Amount = big.NewInt(13)
	SignTestPayment(paymentA, suite.senderPrivateKey)
	paymentB := suite.payment()
	paymentB.Amount = big.NewInt(13)
	SignTestPayment(paymentB, suite.senderPrivateKey)

	transactionA, errA := suite.service.StartPaymentTransaction(paymentA)
	errAC := transactionA.Rollback()
	transactionB, errB := suite.service.StartPaymentTransaction(paymentB)
	errBC := transactionB.Commit()
	channel, ok, errD := suite.storage.Get(suite.channelKey())

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Nil(suite.T(), errAC, "Unexpected error: %v", errAC)
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	assert.Nil(suite.T(), errBC, "Unexpected error: %v", errBC)
	assert.Nil(suite.T(), errD, "Unexpected error: %v", errD)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), suite.channelPlusPayment(paymentB), channel)
}

func (suite *PaymentChannelServiceSuite) TestStartClaim() {
	transaction, _ := suite.service.StartPaymentTransaction(suite.payment())
	transaction.Commit()

	claim, errA := suite.service.StartClaim(suite.channelKey(), IncrementChannelNonce)
	claims, errB := suite.paymentStorage.GetAll()

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	assert.Equal(suite.T(), suite.payment(), claim.Payment())
	assert.Equal(suite.T(), []*Payment{suite.payment()}, claims)
}
