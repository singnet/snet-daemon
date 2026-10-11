package freecall

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func SignFreeTestPayment(payment *Payment, privateKey *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(PrefixSignature),
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
	signerPrivateKey         *ecdsa.PrivateKey
	signerAddress            common.Address
	freeCallUserPrivateKey   *ecdsa.PrivateKey
	freeCallUserAddress      common.Address
	freeCallPaymentValidator PaymentValidator
}

func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}

func (suite *ValidationTestSuite) SetupSuite() {
	suite.signerPrivateKey = GenerateTestPrivateKey()
	suite.signerAddress = crypto.PubkeyToAddress(suite.signerPrivateKey.PublicKey)
	suite.freeCallUserPrivateKey = GenerateTestPrivateKey()
	suite.freeCallUserAddress = crypto.PubkeyToAddress(suite.freeCallUserPrivateKey.PublicKey)
	suite.freeCallPaymentValidator = PaymentValidator{
		freeCallSignerAddress: suite.signerAddress,
		freeCallSigner:        suite.signerPrivateKey,
		currentBlock:          func() (*big.Int, error) { return big.NewInt(99), nil },
	}
}

func (suite *ValidationTestSuite) Payment() *Payment {
	payment := &Payment{
		Address:                    suite.freeCallUserAddress.Hex(),
		ServiceId:                  config.GetString(config.ServiceId),
		OrganizationId:             config.GetString(config.OrganizationId),
		CurrentBlockNumber:         big.NewInt(99),
		AuthTokenExpiryBlockNumber: big.NewInt(120),
		GroupId:                    "default_group",
	}
	var tokenLifetimeBlocks uint64 = 21
	//GenerateFreeCallTokenPayment(payment, suite.signerPrivateKey, suite.userAddress)
	payment.AuthToken, payment.AuthTokenExpiryBlockNumber = suite.freeCallPaymentValidator.NewToken(payment.Address, nil, &tokenLifetimeBlocks)
	tokenParsed, block, err := ParseToken(payment.AuthToken)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), block)
	payment.AuthTokenParsed = tokenParsed
	SignFreeTestPayment(payment, suite.freeCallUserPrivateKey)
	return payment
}

func (suite *ValidationTestSuite) TestFreeCallNewToken() {
	payment := suite.Payment()
	_, deadlineBlock := suite.freeCallPaymentValidator.NewToken(payment.Address, nil, nil)
	assert.NotNil(suite.T(), deadlineBlock, "deadlineBlock can't be nil")
}

func (suite *ValidationTestSuite) TestFreeCallPaymentIsValid() {
	payment := suite.Payment()
	err := suite.freeCallPaymentValidator.Validate(payment)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}
