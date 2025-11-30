package handler

import (
	"context"
	"fmt"

	"github.com/singnet/snet-daemon/v6/codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// WrapperServerStream intercepts gRPC server stream to handle protocol-specific framing
// and header manipulation for dynamic pricing support
type WrapperServerStream struct {
	sendHeaderCalled bool              // Track if headers have been forwarded downstream
	stream           grpc.ServerStream // Original gRPC server stream
	firstMsg         *codec.GrpcFrame  // Initial message read during stream construction
	firstMsgPending  bool              // Flag indicating first message hasn't been delivered via RecvMsg
	Ctx              context.Context   // Context with additional metadata for request processing
}

// NewWrapperServerStream creates a wrapped stream that pre-reads the first message
// and provides a modified context with enhanced metadata
func NewWrapperServerStream(stream grpc.ServerStream, ctx context.Context) (grpc.ServerStream, error) {
	m := &codec.GrpcFrame{}
	if err := stream.RecvMsg(m); err != nil {
		return nil, err
	}
	return &WrapperServerStream{
		stream:          stream,
		firstMsg:        m,
		firstMsgPending: true,
		Ctx:             ctx, // Context now includes additional metadata
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

// SendHeader implements dynamic pricing support by intercepting header transmission
// First call is suppressed (contains backend pricing headers in cogs)
// Subsequent calls are forwarded to the client
func (w *WrapperServerStream) SendHeader(md metadata.MD) error {
	// Dynamic pricing workaround:
	// First call -> suppress (backend headers with pricing in cogs)
	// Second call -> actually transmit to client
	if !w.sendHeaderCalled {
		w.sendHeaderCalled = true
		return nil
	}
	// Subsequent SendHeader calls route to real stream (gRPC prevents multiple calls)
	return w.stream.SendHeader(md)
}

// OriginalRecvMsg provides access to the initially read message for special handling
func (w *WrapperServerStream) OriginalRecvMsg() any {
	return w.firstMsg
}
