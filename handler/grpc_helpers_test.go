package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func setConfigForTest(t *testing.T, key string, value any) {
	t.Helper()
	previous := config.Vip().Get(key)
	t.Cleanup(func() { config.Vip().Set(key, previous) })
	config.Vip().Set(key, value)
}

type coverageTransportStream struct {
	method string
}

func (s coverageTransportStream) Method() string { return s.method }

func (s coverageTransportStream) SetHeader(metadata.MD) error { return nil }

func (s coverageTransportStream) SendHeader(metadata.MD) error { return nil }

func (s coverageTransportStream) SetTrailer(metadata.MD) error { return nil }

func streamContextWithMethod(method string) context.Context {
	return grpc.NewContextWithServerTransportStream(context.Background(), coverageTransportStream{method: method})
}

// coverageServerStream is a controllable grpc.ServerStream.
type coverageServerStream struct {
	ctx           context.Context
	recvQueue     []*codec.GrpcFrame
	sent          []any
	recvErr       error
	sendErr       error
	sendHeaderErr error
	sentHeaders   []metadata.MD
	sentTrailers  []metadata.MD
}

func (s *coverageServerStream) Context() context.Context { return s.ctx }

func (s *coverageServerStream) SetHeader(metadata.MD) error { return nil }

func (s *coverageServerStream) SendHeader(md metadata.MD) error {
	if s.sendHeaderErr != nil {
		return s.sendHeaderErr
	}
	s.sentHeaders = append(s.sentHeaders, md)
	return nil
}

func (s *coverageServerStream) SetTrailer(md metadata.MD) {
	s.sentTrailers = append(s.sentTrailers, md)
}

func (s *coverageServerStream) SendMsg(m any) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, m)
	return nil
}

func (s *coverageServerStream) RecvMsg(m any) error {
	if s.recvErr != nil {
		return s.recvErr
	}
	if len(s.recvQueue) == 0 {
		return io.EOF
	}
	frame, ok := m.(*codec.GrpcFrame)
	if !ok {
		return errors.New("unexpected message type")
	}
	next := s.recvQueue[0]
	s.recvQueue = s.recvQueue[1:]
	if next == nil {
		return io.EOF
	}
	*frame = *next
	return nil
}

var _ grpc.ServerStream = (*coverageServerStream)(nil)

// coverageClientStream is a controllable grpc.ClientStream.
type coverageClientStream struct {
	ctx        context.Context
	recvQueue  []*codec.GrpcFrame
	recvErr    error
	headerErr  error
	closeSend  error
	sendErr    error
	headerMD   metadata.MD
	trailersMD metadata.MD
}

func (s *coverageClientStream) Context() context.Context { return s.ctx }

func (s *coverageClientStream) Trailer() metadata.MD { return s.trailersMD }

func (s *coverageClientStream) CloseSend() error { return s.closeSend }

func (s *coverageClientStream) Header() (metadata.MD, error) {
	if s.headerErr != nil {
		return nil, s.headerErr
	}
	return s.headerMD, nil
}

func (s *coverageClientStream) SendMsg(m any) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	return nil
}

func (s *coverageClientStream) RecvMsg(m any) error {
	if s.recvErr != nil {
		return s.recvErr
	}
	if len(s.recvQueue) == 0 {
		return io.EOF
	}
	frame, ok := m.(*codec.GrpcFrame)
	if !ok {
		return errors.New("unexpected message type")
	}
	next := s.recvQueue[0]
	s.recvQueue = s.recvQueue[1:]
	if next == nil {
		return io.EOF
	}
	*frame = *next
	return nil
}

var _ grpc.ClientStream = (*coverageClientStream)(nil)

func newCoverageMetadata(serviceType, encoding string) *blockchain.ServiceMetadata {
	return &blockchain.ServiceMetadata{
		ServiceType: serviceType,
		Encoding:    encoding,
	}
}

func newCoverageHandler(timeout time.Duration) *grpcHandler {
	return &grpcHandler{
		timeout:    timeout,
		httpClient: &http.Client{Timeout: timeout + time.Second},
		options:    grpc.WithDefaultCallOptions(),
	}
}
