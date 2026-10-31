package handler

import (
	"context"
	"fmt"

	"github.com/singnet/snet-daemon/v6/codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// WrapperServerStream buffers the first raw frame for price estimation, exposes
// an updatable metadata context, and preserves legacy SendHeader suppression.
type WrapperServerStream struct {
	sendHeaderCalled bool              // The first explicit SendHeader call has been suppressed.
	stream           grpc.ServerStream // Original gRPC server stream
	firstMsg         *codec.GrpcFrame  // Initial message read during stream construction
	firstMsgPending  bool              // Flag indicating first message hasn't been delivered via RecvMsg
	Ctx              context.Context   // Context with additional metadata for request processing
}

// NewWrapperServerStream reads the first frame immediately and replays it on the
// first RecvMsg call. Context returns the supplied ctx; it is not enriched here.
func NewWrapperServerStream(stream grpc.ServerStream, ctx context.Context) (grpc.ServerStream, error) {
	m := &codec.GrpcFrame{}
	if err := stream.RecvMsg(m); err != nil {
		return nil, err
	}
	return &WrapperServerStream{
		stream:          stream,
		firstMsg:        m,
		firstMsgPending: true,
		Ctx:             ctx,
	}, nil
}

func (w *WrapperServerStream) Context() context.Context {
	return w.Ctx
}

func (w *WrapperServerStream) RecvMsg(m any) error {
	// First RecvMsg call returns the pre-read frame
	if w.firstMsgPending {
		w.firstMsgPending = false

		dst, ok := m.(*codec.GrpcFrame)
		if !ok {
			return fmt.Errorf("WrapperServerStream: unexpected message type %T, want *codec.GrpcFrame", m)
		}
		*dst = *w.firstMsg
		return nil
	}

	// Subsequent calls delegate to the original stream
	return w.stream.RecvMsg(m)
}

func (w *WrapperServerStream) SendMsg(m any) error {
	return w.stream.SendMsg(m)
}

func (w *WrapperServerStream) SetHeader(md metadata.MD) error {
	return w.stream.SetHeader(md)
}

func (w *WrapperServerStream) SetTrailer(md metadata.MD) {
	w.stream.SetTrailer(md)
}

// SendHeader suppresses its first explicit invocation for compatibility with the
// dynamic-pricing path and forwards later invocations. Suppression is unconditional:
// header contents are not inspected, and SetHeader/SendMsg still delegate normally.
func (w *WrapperServerStream) SendHeader(md metadata.MD) error {
	if !w.sendHeaderCalled {
		w.sendHeaderCalled = true
		return nil
	}
	// Subsequent SendHeader calls route to real stream (gRPC prevents multiple calls)
	return w.stream.SendHeader(md)
}

// OriginalRecvMsg returns the buffered frame without consuming its pending replay.
func (w *WrapperServerStream) OriginalRecvMsg() any {
	return w.firstMsg
}
