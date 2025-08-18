package escrow

import (
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type PaymentHandlerTestSuite struct {
	suite.Suite

	paymentChannelServiceMock PaymentChannelService
	incomeValidatorMock       IncomeStreamValidator

	paymentHandler paymentChannelPaymentHandler
}

func (suite *PaymentHandlerTestSuite) SetupSuite() {
	suite.paymentChannelServiceMock = &paymentChannelServiceMock{
		data: suite.channel(),
		err:  nil,
	}
	suite.incomeValidatorMock = &incomeValidatorMockType{}

	suite.paymentHandler = paymentChannelPaymentHandler{
		service:            suite.paymentChannelServiceMock,
		mpeContractAddress: func() common.Address { return utils.HexToAddress("0xf25186b5081ff5ce73482ad761db0eb0d25abfbf") },
		incomeValidator:    suite.incomeValidatorMock,
	}
}

func TestPaymentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(PaymentHandlerTestSuite))
}

func (suite *PaymentHandlerTestSuite) channel() *PaymentChannelData {
	return &PaymentChannelData{
		AuthorizedAmount: big.NewInt(12300),
	}
}

func (suite *PaymentHandlerTestSuite) grpcMetadata(channelID, channelNonce, amount int64, signature []byte) metadata.MD {
	md := metadata.New(map[string]string{})

	md.Set(handler.PaymentChannelIDHeader, strconv.FormatInt(channelID, 10))
	md.Set(handler.PaymentChannelNonceHeader, strconv.FormatInt(channelNonce, 10))
	md.Set(handler.PaymentChannelAmountHeader, strconv.FormatInt(amount, 10))
	md.Set(handler.PaymentChannelSignatureHeader, string(signature))

	return md
}

func (suite *PaymentHandlerTestSuite) grpcContext(patch func(*metadata.MD)) *handler.GrpcStreamContext {
	md := suite.grpcMetadata(42, 3, 12345, []byte{0x1, 0x2, 0xFE, 0xFF})
	patch(&md)
	return &handler.GrpcStreamContext{
		MD: md,
	}
}

func (suite *PaymentHandlerTestSuite) TestGetPayment() {
	context := suite.grpcContext(func(md *metadata.MD) {})

	_, err := suite.paymentHandler.Payment(context)

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *PaymentHandlerTestSuite) TestGetPaymentNoChannelId() {
	context := suite.grpcContext(func(md *metadata.MD) {
		delete(*md, handler.PaymentChannelIDHeader)
	})

	payment, err := suite.paymentHandler.Payment(context)

	assert.Equal(suite.T(), handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-id\""), err)
	assert.Nil(suite.T(), payment)
}

func (suite *PaymentHandlerTestSuite) TestGetPaymentNoChannelNonce() {
	context := suite.grpcContext(func(md *metadata.MD) {
		delete(*md, handler.PaymentChannelNonceHeader)
	})

	payment, err := suite.paymentHandler.Payment(context)

	assert.Equal(suite.T(), handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-nonce\""), err)
	assert.Nil(suite.T(), payment)
}

func (suite *PaymentHandlerTestSuite) TestGetPaymentNoChannelAmount() {
	context := suite.grpcContext(func(md *metadata.MD) {
		delete(*md, handler.PaymentChannelAmountHeader)
	})

	payment, err := suite.paymentHandler.Payment(context)

	assert.Equal(suite.T(), handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-amount\""), err)
	assert.Nil(suite.T(), payment)
}

func (suite *PaymentHandlerTestSuite) TestGetPaymentNoSignature() {
	context := suite.grpcContext(func(md *metadata.MD) {
		delete(*md, handler.PaymentChannelSignatureHeader)
	})

	payment, err := suite.paymentHandler.Payment(context)

	assert.Equal(suite.T(), handler.NewGrpcError(codes.InvalidArgument, "missing \"snet-payment-channel-signature-bin\""), err)
	assert.Nil(suite.T(), payment)
}

func (suite *PaymentHandlerTestSuite) TestStartTransactionError() {
	context := suite.grpcContext(func(md *metadata.MD) {})
	paymentHandler := suite.paymentHandler
	paymentHandler.service = &paymentChannelServiceMock{
		err: NewPaymentError(FailedPrecondition, "another transaction in progress"),
	}

	payment, err := paymentHandler.Payment(context)

	assert.Equal(suite.T(), handler.NewGrpcError(codes.FailedPrecondition, "another transaction in progress"), err)
	assert.Nil(suite.T(), payment)
}

func (suite *PaymentHandlerTestSuite) TestValidatePaymentIncorrectIncome() {
	context := suite.grpcContext(func(md *metadata.MD) {})
	incomeErr := NewPaymentError(Unauthenticated, "incorrect payment income: \"45\", expected \"46\"")
	paymentHandler := suite.paymentHandler
	paymentHandler.incomeValidator = &incomeValidatorMockType{err: incomeErr}

	payment, err := paymentHandler.Payment(context)

	assert.Equal(suite.T(), handler.NewGrpcError(codes.Unauthenticated, "incorrect payment income: \"45\", expected \"46\""), err)
	assert.Nil(suite.T(), payment)
}

func Test_paymentChannelPaymentHandler_PublishChannelStats(t *testing.T) {
	mocked := blockchain.NewMockProcessor(true)
	payment := &paymentTransaction{payment: Payment{Amount: big.NewInt(10), ChannelID: big.NewInt(6),
		ChannelNonce: big.NewInt(0)}, channel: &PaymentChannelData{FullAmount: big.NewInt(10), Nonce: big.NewInt(0)}}
	tests := []struct {
		name string

		wantErr   *handler.GrpcError
		setupFunc func()
	}{
		{name: "disabled metering", wantErr: nil, setupFunc: func() {
		}},

		{name: "", wantErr: nil, setupFunc: func() {
			config.Vip().Set(config.OrganizationId, "ExampleOrganizationId")
			config.Vip().Set(config.ServiceId, "ExampleServiceId")
			config.Vip().Set(config.MeteringEnabled, true)
			config.Vip().Set(config.PvtKeyForMetering, "063C00D18E147F4F734846E47FE6598FC7A6D56307862F7EDC92B9F43CC27EDD")
			config.Vip().Set(config.MeteringEndpoint, "https://bkq2d3zjl4.execute-api.eu-west-1.amazonaws.com/main")
		}},

		{name: "", wantErr: handler.NewGrpcErrorf(codes.Internal, "Unable to publish status error"), setupFunc: func() {
			config.Vip().Set(config.MeteringEndpoint, "badurl")
		}},
	}
	for _, tt := range tests {
		tt.setupFunc()
		t.Run(tt.name, func(t *testing.T) {
			if gotErr := PublishChannelStats(payment, mocked.CurrentBlock); !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("paymentChannelPaymentHandler.PublishChannelStats() = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}
