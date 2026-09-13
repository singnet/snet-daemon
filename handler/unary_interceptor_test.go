package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/ctxkeys"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type unaryPaymentMock struct {
	sender common.Address
}

func (payment unaryPaymentMock) GetSender() common.Address {
	return payment.sender
}

type unaryPaymentHandlerMock struct {
	typ                      string
	payment                  Payment
	paymentErr               *GrpcError
	completeErr              *GrpcError
	completeAfterErrorErr    *GrpcError
	completeCalled           bool
	completeAfterErrorCalled bool
}

func (handler *unaryPaymentHandlerMock) Type() string {
	return handler.typ
}

func (handler *unaryPaymentHandlerMock) Payment(*GrpcUnaryContext) (Payment, *GrpcError) {
	return handler.payment, handler.paymentErr
}

func (handler *unaryPaymentHandlerMock) Complete(Payment) *GrpcError {
	handler.completeCalled = true
	return handler.completeErr
}

func (handler *unaryPaymentHandlerMock) CompleteAfterError(Payment, error) *GrpcError {
	handler.completeAfterErrorCalled = true
	return handler.completeAfterErrorErr
}

func TestUnaryPaymentValidationInterceptorCompletesTrainingPayment(t *testing.T) {
	sender := common.HexToAddress("0x00000000000000000000000000000000000000ab")
	paymentHandler := &unaryPaymentHandlerMock{typ: "train", payment: unaryPaymentMock{sender: sender}}
	interceptor := GrpcPaymentValidationUnaryInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(PaymentTypeHeader, "train"))
	info := &grpc.UnaryServerInfo{FullMethod: "/training.Service/train_model"}

	response, err := interceptor(ctx, "request", info, func(ctx context.Context, request any) (any, error) {
		method, ok := ctx.Value(ctxkeys.MethodKey).(string)
		require.True(t, ok)
		require.Equal(t, info.FullMethod, method)
		metadataFromHandler, ok := metadata.FromIncomingContext(ctx)
		require.True(t, ok)
		require.Equal(t, sender.Hex(), metadataFromHandler.Get(SnetUserAddressHeader)[0])
		return "response", nil
	})

	require.NoError(t, err)
	require.Equal(t, "response", response)
	require.True(t, paymentHandler.completeCalled)
	require.False(t, paymentHandler.completeAfterErrorCalled)
}

func TestUnaryPaymentValidationInterceptorCompletesAfterHandlerError(t *testing.T) {
	paymentHandler := &unaryPaymentHandlerMock{typ: "train", payment: unaryPaymentMock{}}
	interceptor := GrpcPaymentValidationUnaryInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(PaymentTypeHeader, "train"))
	expected := errors.New("service failure")

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/training.Service/train_model"}, func(context.Context, any) (any, error) {
		return nil, expected
	})

	require.ErrorIs(t, err, expected)
	require.False(t, paymentHandler.completeCalled)
	require.True(t, paymentHandler.completeAfterErrorCalled)
}

func TestUnaryPaymentValidationInterceptorBypassesNonTrainingRequests(t *testing.T) {
	paymentHandler := &unaryPaymentHandlerMock{typ: "train", payment: unaryPaymentMock{}}
	interceptor := GrpcPaymentValidationUnaryInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	info := &grpc.UnaryServerInfo{FullMethod: "/example.Service/ping"}

	response, err := interceptor(ctx, "request", info, func(ctx context.Context, request any) (any, error) {
		require.Equal(t, info.FullMethod, ctx.Value(ctxkeys.MethodKey))
		return "response", nil
	})

	require.NoError(t, err)
	require.Equal(t, "response", response)
	require.False(t, paymentHandler.completeCalled)
	require.False(t, paymentHandler.completeAfterErrorCalled)
}

func TestUnaryPaymentValidationInterceptorRejectsMissingOrUnknownPaymentMetadata(t *testing.T) {
	paymentHandler := &unaryPaymentHandlerMock{typ: "train", payment: unaryPaymentMock{}}
	interceptor := GrpcPaymentValidationUnaryInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
	info := &grpc.UnaryServerInfo{FullMethod: "/training.Service/train_model"}

	_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) { return nil, nil })
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	unknownTypeCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(PaymentTypeHeader, "unknown"))
	_, err = interceptor(unknownTypeCtx, nil, info, func(context.Context, any) (any, error) { return nil, nil })
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.False(t, paymentHandler.completeCalled)
}

func TestSetAdditionalDetailsCopiesOptionalMetadata(t *testing.T) {
	context := &GrpcStreamContext{MD: metadata.Pairs(
		ClientTypeHeader, "sdk",
		UserInfoHeader, "client",
		UserAgentHeader, "agent",
		PaymentChannelIDHeader, "42",
		FreeCallUserIdHeader, "user",
		PaymentTypeHeader, "escrow",
	)}
	stats := &metrics.CommonStats{}

	setAdditionalDetails(context, stats)

	require.Equal(t, "sdk", stats.ClientType)
	require.Equal(t, "client", stats.UserDetails)
	require.Equal(t, "agent", stats.UserAgent)
	require.Equal(t, "42", stats.ChannelId)
	require.Equal(t, "user", stats.UserName)
	require.Equal(t, "escrow", stats.PaymentMode)
}
