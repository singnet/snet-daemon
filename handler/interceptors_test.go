package handler

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	defaultPaymentHandlerType = "test-default-payment-handler"
	testPaymentHandlerType    = "test-payment-handler"
)

type paymentMock struct {
}

type paymentHandlerMock struct {
	typ                      string
	completeAfterErrorCalled bool
	completeCalled           bool
	completeResult           *GrpcError
	completeAfterErrorResult *GrpcError
	paymentResult            *GrpcError
	payment                  *paymentMock
}

func (handler *paymentHandlerMock) reset() {
	handler.completeAfterErrorCalled = false
	handler.completeCalled = false
	handler.completeResult = nil
	handler.completeAfterErrorResult = nil
	handler.paymentResult = nil
	handler.payment = nil
}

func (handler *paymentHandlerMock) Type() string {
	return handler.typ
}

func (handler *paymentHandlerMock) Payment(context *GrpcStreamContext) (payment Payment, err *GrpcError) {
	if handler.paymentResult != nil {
		return nil, handler.paymentResult
	}
	handler.payment = &paymentMock{}
	return handler.payment, nil
}

func (handler *paymentHandlerMock) Complete(payment Payment) (err *GrpcError) {
	handler.completeCalled = true
	if payment != handler.payment {
		return NewGrpcError(codes.Internal, "invalid payment")
	}
	return handler.completeResult
}

func (handler *paymentHandlerMock) CompleteAfterError(payment Payment, result error) (err *GrpcError) {
	handler.completeAfterErrorCalled = true
	if payment != handler.payment {
		return NewGrpcError(codes.Internal, "invalid payment")
	}
	return handler.completeAfterErrorResult
}

type InterceptorsSuite struct {
	suite.Suite

	successHandler        grpc.StreamHandler
	returnErrorHandler    grpc.StreamHandler
	panicHandler          grpc.StreamHandler
	defaultPaymentHandler *paymentHandlerMock
	paymentHandler        *paymentHandlerMock
	interceptor           grpc.StreamServerInterceptor
	serverStream          *serverStreamMock
}

func (suite *InterceptorsSuite) SetupSuite() {
	suite.successHandler = func(srv any, stream grpc.ServerStream) error {
		return nil
	}
	suite.returnErrorHandler = func(srv any, stream grpc.ServerStream) error {
		return errors.New("some error")
	}
	suite.panicHandler = func(srv any, stream grpc.ServerStream) error {
		panic("some panic")
	}
	suite.defaultPaymentHandler = &paymentHandlerMock{typ: defaultPaymentHandlerType}
	suite.paymentHandler = &paymentHandlerMock{typ: testPaymentHandlerType}
	suite.interceptor = GrpcPaymentValidationInterceptor(&blockchain.ServiceMetadata{}, suite.defaultPaymentHandler, suite.paymentHandler)
	suite.serverStream = &serverStreamMock{context: metadata.NewIncomingContext(context.Background(), metadata.Pairs(PaymentTypeHeader, testPaymentHandlerType))}
}

func (suite *InterceptorsSuite) SetupTest() {
	suite.paymentHandler.reset()
}

func TestInterceptorsSuite(t *testing.T) {
	suite.Run(t, new(InterceptorsSuite))
}

func (suite *InterceptorsSuite) TestGetBytesFromHexString() {
	md := metadata.Pairs("test-key", "0xfFfE0100")

	bytes, err := GetBytesFromHex(md, "test-key")

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), []byte{255, 254, 1, 0}, bytes)
}

func (suite *InterceptorsSuite) TestGetBytesFromHexStringNoPrefix() {
	md := metadata.Pairs("test-key", "fFfE0100")

	bytes, err := GetBytesFromHex(md, "test-key")

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), []byte{255, 254, 1, 0}, bytes)
}

func (suite *InterceptorsSuite) TestGetBytesFromHexStringNoValue() {
	md := metadata.Pairs("unknown-key", "fFfE0100")

	_, err := GetBytesFromHex(md, "test-key")

	assert.Equal(suite.T(), NewGrpcErrorf(codes.InvalidArgument, "missing \"test-key\""), err)
}

func (suite *InterceptorsSuite) TestGetBytesFromHexStringTooManyValues() {
	md := metadata.Pairs("test-key", "0x123", "test-key", "FED")

	_, err := GetBytesFromHex(md, "test-key")

	assert.Equal(suite.T(), NewGrpcErrorf(codes.InvalidArgument, "too many values for key \"test-key\": [0x123 FED]"), err)
}

func (suite *InterceptorsSuite) TestGetBigInt() {
	md := metadata.Pairs("big-int-key", "12345")

	value, err := GetBigInt(md, "big-int-key")

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), big.NewInt(12345), value)
}

func (suite *InterceptorsSuite) TestGetBigIntIncorrectValue() {
	md := metadata.Pairs("big-int-key", "12345abc")

	_, err := GetBigInt(md, "big-int-key")

	assert.Equal(suite.T(), NewGrpcErrorf(codes.InvalidArgument, "incorrect format \"big-int-key\": \"12345abc\""), err)
}

func (suite *InterceptorsSuite) TestGetBigIntNoValue() {
	md := metadata.Pairs()

	_, err := GetBigInt(md, "big-int-key")

	assert.Equal(suite.T(), NewGrpcErrorf(codes.InvalidArgument, "missing \"big-int-key\""), err)
}

func (suite *InterceptorsSuite) TestGetBigIntTooManyValues() {
	md := metadata.Pairs("big-int-key", "12345", "big-int-key", "54321")

	_, err := GetBigInt(md, "big-int-key")

	assert.Equal(suite.T(), NewGrpcErrorf(codes.InvalidArgument, "too many values for key \"big-int-key\": [12345 54321]"), err)
}

func (suite *InterceptorsSuite) TestGetBytes() {
	md := metadata.Pairs("binary-key-bin", string([]byte{0x00, 0x01, 0xFE, 0xFF}))

	value, err := GetBytes(md, "binary-key-bin")

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), []byte{0, 1, 254, 255}, value)
}

func (suite *InterceptorsSuite) TestGetBytesIncorrectBinaryKey() {
	md := metadata.Pairs("binary-key", string([]byte{0x00, 0x01, 0xFE, 0xFF}))

	_, err := GetBytes(md, "binary-key")

	assert.Equal(suite.T(), NewGrpcErrorf(codes.InvalidArgument, "incorrect binary key name \"binary-key\""), err)
}

func (suite *InterceptorsSuite) TestCompleteOnHandlerError() {
	suite.interceptor(nil, suite.serverStream, nil, suite.returnErrorHandler)

	assert.True(suite.T(), suite.paymentHandler.completeAfterErrorCalled)
	assert.False(suite.T(), suite.paymentHandler.completeCalled)
}

func (suite *InterceptorsSuite) TestCompleteOnHandlerPanic() {
	defer func() {
		if r := recover(); r == nil {
			assert.Fail(suite.T(), "panic() call expected")
		}
	}()

	suite.interceptor(nil, suite.serverStream, nil, suite.panicHandler)

	assert.True(suite.T(), suite.paymentHandler.completeAfterErrorCalled)
	assert.False(suite.T(), suite.paymentHandler.completeCalled)
}

func (suite *InterceptorsSuite) TestCompleteOnHandlerSuccess() {
	suite.interceptor(nil, suite.serverStream, nil, suite.successHandler)

	assert.True(suite.T(), suite.paymentHandler.completeCalled)
	assert.False(suite.T(), suite.paymentHandler.completeAfterErrorCalled)
}

func (suite *InterceptorsSuite) TestCompleteReturnsError() {
	suite.paymentHandler.completeResult = NewGrpcError(codes.Internal, "test error")

	err := suite.interceptor(nil, suite.serverStream, nil, suite.successHandler)

	assert.Equal(suite.T(), status.Newf(codes.Internal, "test error").Err(), err)
}

func (suite *InterceptorsSuite) TestCompleteAfterErrorReturnsError() {
	suite.paymentHandler.completeAfterErrorResult = NewGrpcError(codes.Internal, "test error")

	err := suite.interceptor(nil, suite.serverStream, nil, suite.returnErrorHandler)
	assert.Error(suite.T(), err)
}

func (suite *InterceptorsSuite) TestPaymentReturnsError() {
	suite.paymentHandler.paymentResult = NewGrpcError(codes.Internal, "test error")

	err := suite.interceptor(nil, suite.serverStream, nil, suite.successHandler)

	assert.Equal(suite.T(), status.Newf(codes.Internal, "test error").Err(), err)
}
