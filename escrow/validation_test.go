package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func SignTestPayment(payment *Payment, privateKey *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	payment.Signature = getSignature(message, privateKey)
}

func SignFreeTestPayment(payment *FreeCallPayment, privateKey *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(payment.Address),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(payment.GroupId),
		bigIntToBytes(payment.CurrentBlockNumber),
		payment.AuthToken,
	}, nil)

	payment.Signature = getSignature(message, privateKey)
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

func GenerateTestPrivateKey() (privateKey *ecdsa.PrivateKey) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Cannot generate private key for test: %v", err))
	}
	return
}

type ValidationTestSuite struct {
	suite.Suite

	senderAddress          common.Address
	signerPrivateKey       *ecdsa.PrivateKey
	signerAddress          common.Address
	recipientAddress       common.Address
	mpeContractAddress     common.Address
	freeCallUserPrivateKey *ecdsa.PrivateKey
	freeCallUserAddress    common.Address

	validator                ChannelPaymentValidator
	freeCallPaymentValidator FreeCallPaymentValidator
}

func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}

func (suite *ValidationTestSuite) SetupSuite() {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	suite.senderAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.signerPrivateKey = GenerateTestPrivateKey()
	suite.freeCallUserPrivateKey = GenerateTestPrivateKey()

	suite.signerAddress = crypto.PubkeyToAddress(suite.signerPrivateKey.PublicKey)
	suite.freeCallUserAddress = crypto.PubkeyToAddress(suite.freeCallUserPrivateKey.PublicKey)
	suite.recipientAddress = crypto.PubkeyToAddress(GenerateTestPrivateKey().PublicKey)
	suite.mpeContractAddress = utils.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf")

	suite.validator = ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
	}
	suite.freeCallPaymentValidator = FreeCallPaymentValidator{freeCallSignerAddress: suite.signerAddress, freeCallSigner: suite.signerPrivateKey,
		currentBlock: func() (*big.Int, error) { return big.NewInt(99), nil }}
}

func (suite *ValidationTestSuite) FreeCallPayment() *FreeCallPayment {
	payment := &FreeCallPayment{
		Address:                    suite.freeCallUserAddress.Hex(),
		ServiceId:                  config.GetString(config.ServiceId),
		OrganizationId:             config.GetString(config.OrganizationId),
		CurrentBlockNumber:         big.NewInt(99),
		AuthTokenExpiryBlockNumber: big.NewInt(120),
		GroupId:                    "default_group",
	}
	var tokenLifetimeBlocks uint64 = 21
	//GenerateFreeCallTokenPayment(payment, suite.signerPrivateKey, suite.userAddress)
	payment.AuthToken, payment.AuthTokenExpiryBlockNumber = suite.freeCallPaymentValidator.NewFreeCallToken(payment.Address, nil, &tokenLifetimeBlocks)
	tokenParsed, block, err := ParseFreeCallToken(payment.AuthToken)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), block)
	payment.AuthTokenParsed = tokenParsed
	SignFreeTestPayment(payment, suite.freeCallUserPrivateKey)
	return payment
}

func (suite *ValidationTestSuite) payment() *Payment {
	payment := &Payment{
		Amount:             big.NewInt(12345),
		ChannelID:          big.NewInt(42),
		ChannelNonce:       big.NewInt(3),
		MpeContractAddress: suite.mpeContractAddress,
	}
	SignTestPayment(payment, suite.signerPrivateKey)
	return payment
}

func (suite *ValidationTestSuite) channel() *PaymentChannelData {
	return &PaymentChannelData{
		ChannelID:        big.NewInt(42),
		Nonce:            big.NewInt(3),
		Sender:           suite.senderAddress,
		Recipient:        suite.recipientAddress,
		GroupID:          [32]byte{123},
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		Signer:           suite.signerAddress,
		AuthorizedAmount: big.NewInt(12300),
		Signature:        nil,
	}
}

func (suite *ValidationTestSuite) TestFreeCallNewToken() {
	payment := suite.FreeCallPayment()
	_, deadlineBlock := suite.freeCallPaymentValidator.NewFreeCallToken(payment.Address, nil, nil)
	assert.NotNil(suite.T(), deadlineBlock, "deadlineBlock can't be nil")
}

func (suite *ValidationTestSuite) TestFreeCallPaymentIsValid() {
	payment := suite.FreeCallPayment()
	err := suite.freeCallPaymentValidator.Validate(payment)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *ValidationTestSuite) TestPaymentIsValid() {
	payment := suite.payment()
	channel := suite.channel()

	err := suite.validator.Validate(payment, channel)

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *ValidationTestSuite) TestValidatePaymentChannelNonce() {
	payment := suite.payment()
	payment.ChannelNonce = big.NewInt(2)
	SignTestPayment(payment, suite.signerPrivateKey)
	channel := suite.channel()
	channel.Nonce = big.NewInt(3)

	err := suite.validator.Validate(payment, channel)

	assert.Equal(suite.T(), NewPaymentError(IncorrectNonce, "incorrect payment channel nonce, latest: 3, sent: 2"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentIncorrectSignatureLength() {
	payment := suite.payment()
	payment.Signature = utils.HexToBytes("0x0000")

	err := suite.validator.Validate(payment, suite.channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment signature is not valid"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentIncorrectSignatureChecksum() {
	payment := suite.payment()
	payment.Signature = utils.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef21")

	err := suite.validator.Validate(payment, suite.channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment signature is not valid"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentIncorrectSigner() {
	payment := suite.payment()
	payment.Signature = utils.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01")

	err := suite.validator.Validate(payment, suite.channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment is not signed by channel signer/sender"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentChannelCannotGetCurrentBlock() {
	validator := &ChannelPaymentValidator{
		currentBlock: func() (*big.Int, error) { return nil, errors.New("blockchain error") },
	}

	err := validator.Validate(suite.payment(), suite.channel())

	assert.Equal(suite.T(), NewPaymentError(Internal, "cannot determine current block"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentExpiredChannel() {
	validator := &ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
	}
	channel := suite.channel()
	channel.Expiration = big.NewInt(99)

	err := validator.Validate(suite.payment(), channel)

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 99, expiration threshold: 0"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentChannelExpirationThreshold() {
	validator := &ChannelPaymentValidator{
		currentBlock:               func() (*big.Int, error) { return big.NewInt(98), nil },
		paymentExpirationThreshold: func() *big.Int { return big.NewInt(1) },
	}
	channel := suite.channel()
	channel.Expiration = big.NewInt(99)

	err := validator.Validate(suite.payment(), channel)

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: 99, current block: 98, expiration threshold: 1"), err)
}

func (suite *ValidationTestSuite) TestValidatePaymentAmountIsTooBig() {
	payment := suite.payment()
	payment.Amount = big.NewInt(12346)
	SignTestPayment(payment, suite.signerPrivateKey)
	channel := suite.channel()
	channel.FullAmount = big.NewInt(12345)

	err := suite.validator.Validate(payment, suite.channel())

	assert.Equal(suite.T(), NewPaymentError(Unauthenticated, "not enough tokens on payment channel, channel amount: 12345, payment amount: 12346"), err)
}

func (suite *ValidationTestSuite) TestGetPublicKeyFromPayment() {
	payment := Payment{
		MpeContractAddress: suite.mpeContractAddress,
		ChannelID:          big.NewInt(1789),
		ChannelNonce:       big.NewInt(1917),
		Amount:             big.NewInt(31415),
		// message hash: 04cc38aa4a27976907ef7382182bc549957dc9d2e21eb73651ad6588d5cd4d8f
		Signature: utils.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	}

	address, err := getSignerAddressFromPayment(&payment)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.Equal(suite.T(), utils.HexToAddress("0x77D524c6e0FD652aA9A9bFcAd1d92Fe0781767dF"), *address)
}

func (suite *ValidationTestSuite) TestGetPublicKeyFromPayment2() {
	payment := Payment{
		MpeContractAddress: utils.HexToAddress("0x39ee715b50e78a920120c1ded58b1a47f571ab75"),
		ChannelID:          big.NewInt(1789),
		ChannelNonce:       big.NewInt(1917),
		Amount:             big.NewInt(31415),
		Signature:          utils.HexToBytes("0xde4e998341307b036e460b1cc1593ddefe2e9ea261bd6c3d75967b29b2c3d0a24969b4a32b099ae2eded90bbc213ad0a159a66af6d55be7e04f724ffa52ce3cc1b"),
	}

	address, err := getSignerAddressFromPayment(&payment)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), utils.HexToAddress("0x6b1E951a2F9dE2480C613C1dCDDee4DD4CaE1e4e"), *address)
}
