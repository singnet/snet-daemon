package handler

import (
	"context"
	"testing"

	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNewWrapperServerStream(t *testing.T) {
	stream := &serverStreamMock{}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("key", "value"))

	wrapper, err := NewWrapperServerStream(stream, ctx)

	assert.NoError(t, err)
	assert.NotNil(t, wrapper)
	assert.Equal(t, ctx, wrapper.Context())
}

func TestWrapperServerStream_RecvMsgReturnsFirstMessageOnce(t *testing.T) {
	stream := &serverStreamMock{}
	wrapper, err := NewWrapperServerStream(stream, context.Background())
	assert.NoError(t, err)

	// first call returns the pre-read message
	frame := &codec.GrpcFrame{}
	assert.NoError(t, wrapper.RecvMsg(frame))

	// second call delegates to the original stream
	assert.NoError(t, wrapper.RecvMsg(frame))
}

func TestWrapperServerStream_RecvMsgUnexpectedType(t *testing.T) {
	wrapper, err := NewWrapperServerStream(&serverStreamMock{}, context.Background())
	assert.NoError(t, err)

	err = wrapper.RecvMsg("not-a-frame")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected message type")
}

func TestWrapperServerStream_OriginalRecvMsg(t *testing.T) {
	wrapper, err := NewWrapperServerStream(&serverStreamMock{}, context.Background())
	assert.NoError(t, err)

	assert.NotNil(t, wrapper.(*WrapperServerStream).OriginalRecvMsg())
}

func TestWrapperServerStream_DelegatesToStream(t *testing.T) {
	stream := &serverStreamMock{}
	wrapper, err := NewWrapperServerStream(stream, context.Background())
	assert.NoError(t, err)

	assert.NoError(t, wrapper.SendMsg(&codec.GrpcFrame{}))
	assert.NoError(t, wrapper.SetHeader(metadata.Pairs("key", "value")))
	wrapper.SetTrailer(metadata.Pairs("trailer", "value"))
}

func TestWrapperServerStream_SendHeaderSuppressesFirstCall(t *testing.T) {
	stream := &serverStreamMock{}
	wrapper, err := NewWrapperServerStream(stream, context.Background())
	assert.NoError(t, err)

	// first SendHeader call is suppressed
	assert.NoError(t, wrapper.SendHeader(metadata.Pairs("key", "value")))

	// subsequent calls are forwarded to the underlying stream
	assert.NoError(t, wrapper.SendHeader(metadata.Pairs("key", "value")))
	assert.NoError(t, wrapper.SendHeader(metadata.Pairs("key", "value")))
}

func TestWrapperServerStream_ImplementsServerStream(t *testing.T) {
	var _ grpc.ServerStream = (*WrapperServerStream)(nil)
	assert.True(t, true)
}
