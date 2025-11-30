package handler

import (
	"context"

	"github.com/singnet/snet-daemon/v6/codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type WrapperServerStream struct {
	sendHeaderCalled bool
	stream           grpc.ServerStream
	recvMessage      any
	sentMessage      any
	Ctx              context.Context
}

func (f *WrapperServerStream) SetTrailer(md metadata.MD) {
	f.stream.SetTrailer(md)
}

func NewWrapperServerStream(stream grpc.ServerStream, ctx context.Context) (grpc.ServerStream, error) {
	m := &codec.GrpcFrame{}
	err := stream.RecvMsg(m)
	f := &WrapperServerStream{
		stream:           stream,
		recvMessage:      m,
		sendHeaderCalled: false,
		Ctx:              ctx, // save modified ctx
	}
	return f, err
}

func (f *WrapperServerStream) Context() context.Context {
	// old way return f.stream.Context()
	return f.Ctx // return modified context
}

func (f *WrapperServerStream) SetHeader(md metadata.MD) error {
	return f.stream.SetHeader(md)
}

func (f *WrapperServerStream) SendHeader(md metadata.MD) error {
	//this is more of a hack to support dynamic pricing
	// when the service method returns the price in cogs, the SendHeader will be called,
	// we don't want this as the SendHeader can be called just once in the ServerStream
	if !f.sendHeaderCalled {
		return nil
	}
	f.sendHeaderCalled = true
	return f.stream.SendHeader(md)
}

func (f *WrapperServerStream) SendMsg(m any) error {
	return f.stream.SendMsg(m)
}

func (f *WrapperServerStream) RecvMsg(m any) error {
	return f.stream.RecvMsg(m)
}

func (f *WrapperServerStream) OriginalRecvMsg() any {
	return f.recvMessage
}
