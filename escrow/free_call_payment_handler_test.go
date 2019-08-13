package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

type FreeCallPaymentHandlerTestSuite struct {
	suite.Suite
	paymentHandler freeCallPaymentHandler
	privateKey     *ecdsa.PrivateKey
}

func (suite *FreeCallPaymentHandlerTestSuite) SetupSuite() {

	suite.privateKey = GenerateTestPrivateKey()

	suite.paymentHandler = freeCallPaymentHandler{
		freeCallPaymentValidator: NewFreeCallPaymentValidator(func() (*big.Int, error) {
			return big.NewInt(99), nil
		}, crypto.PubkeyToAddress(suite.privateKey.PublicKey)),
	}
	config.Vip().Set(config.MeteringEndPoint,"http://demo8325345.mockable.io")
}

func SignTestFreeCallPayment(privateKey *ecdsa.PrivateKey, currentBlock int64, user string) []byte {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(user),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		bigIntToBytes(big.NewInt(currentBlock)),
	}, nil)

	return getSignature(message, privateKey)
}

func TestFreeCallPaymentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(FreeCallPaymentHandlerTestSuite))
}

func (suite *FreeCallPaymentHandlerTestSuite) grpcMetadataForFreeCall(user string, currentBlock int64) metadata.MD {
	md := metadata.New(map[string]string{})
	md.Set(handler.FreeCallUserIdHeader, user)
	md.Set(handler.CurrentBlockNumberHeader, strconv.FormatInt(currentBlock, 10))
	md.Set(handler.PaymentChannelSignatureHeader, string(SignTestFreeCallPayment(suite.privateKey, currentBlock, user)))
	println("************************************************************")
	println((SignTestFreeCallPayment(suite.privateKey, currentBlock, user)))
	println("************************************************************")
	return md
}

func (suite *FreeCallPaymentHandlerTestSuite) grpcContextForFreeCall(patch func(*metadata.MD)) *handler.GrpcStreamContext {
	md := suite.grpcMetadataForFreeCall("user1", 99)
	patch(&md)
	return &handler.GrpcStreamContext{
		MD: md,
	}
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPayment() {
	context := suite.grpcContextForFreeCall(func(md *metadata.MD) {})
	_, err := suite.paymentHandler.Payment(context)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *FreeCallPaymentHandlerTestSuite) Test_freeCallPaymentHandler_Type() {
	assert.Equal(suite.T(), suite.paymentHandler.Type(), FreeCallPaymentType)
}

func Test_checkResponse(t *testing.T) {
	response, err := sendRequest(nil,
		"http://demo8325345.mockable.io/usage/freecalls","testuser")
	assert.NotNil(t,response)
	assert.Nil(t,err)
	allowed , err := checkResponse(response)
	assert.True(t,allowed)
}
