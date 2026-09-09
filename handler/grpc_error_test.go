package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGrpcError_ErrorAndString(t *testing.T) {
	err := NewGrpcError(codes.Internal, "test error")

	assert.Equal(t, "{Status: rpc error: code = Internal desc = test error}", err.Error())
	assert.Equal(t, "{Status: "+err.Status.String()+"}", err.String())
}

func TestGrpcError_Err(t *testing.T) {
	err := NewGrpcError(codes.Internal, "test error")

	assert.Equal(t, status.New(codes.Internal, "test error").Err(), err.Err())
}

func TestGrpcError_ErrNilStatus(t *testing.T) {
	err := &GrpcError{}

	assert.Nil(t, err.Err())
}

func TestNewGrpcErrorf(t *testing.T) {
	// with args
	err := NewGrpcErrorf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", "key", "value")
	assert.Equal(t, status.Newf(codes.InvalidArgument, "unexpected \"key\", value: \"value\"").Err(), err.Err())

	// without args the format string is used as is
	err = NewGrpcErrorf(codes.InvalidArgument, "some message")
	assert.Equal(t, status.New(codes.InvalidArgument, "some message").Err(), err.Err())
}

func TestGrpcStreamContext_String(t *testing.T) {
	grpcContext := &GrpcStreamContext{
		MD:   metadata.Pairs("key", "value"),
		Info: &grpc.StreamServerInfo{FullMethod: "/test.Service/method"},
	}

	assert.Contains(t, grpcContext.String(), "key:[value]")
	assert.Contains(t, grpcContext.String(), "/test.Service/method")
}

func TestNoOpInterceptor(t *testing.T) {
	handlerCalled := false
	err := NoOpInterceptor(nil, &serverStreamMock{}, &grpc.StreamServerInfo{}, func(srv any, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestNoOpUnaryInterceptor(t *testing.T) {
	handlerCalled := false
	ctx := context.Background()
	resp, err := NoOpUnaryInterceptor(ctx, "request", &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "response", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)
}

func TestWithDefaultTimeout_NoDeadlineSet(t *testing.T) {
	ctx, cancel := withDefaultTimeout(context.Background(), time.Minute)
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, time.Second)
}

func TestWithDefaultTimeout_DeadlineAlreadySet(t *testing.T) {
	initialDeadline := time.Now().Add(2 * time.Minute)
	initialCtx, initialCancel := context.WithDeadline(context.Background(), initialDeadline)
	defer initialCancel()

	ctx, cancel := withDefaultTimeout(initialCtx, time.Minute)
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, initialDeadline, deadline, time.Second)
}

func TestWithDefaultTimeout_NonPositiveDuration(t *testing.T) {
	ctx, cancel := withDefaultTimeout(context.Background(), 0)
	defer cancel()

	_, ok := ctx.Deadline()
	assert.False(t, ok)
}

func TestCors(t *testing.T) {
	assert.NotNil(t, Cors())
}
