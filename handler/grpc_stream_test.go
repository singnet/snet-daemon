package handler

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestForwardClientToServer_RecvError(t *testing.T) {
	src := &coverageClientStream{recvErr: errors.New("recv failed")}
	dst := &coverageServerStream{ctx: context.Background()}

	err := <-forwardClientToServer(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recv failed")
}

func TestForwardClientToServer_HeaderError(t *testing.T) {
	src := &coverageClientStream{
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A}}},
		headerErr: errors.New("header failed"),
	}
	dst := &coverageServerStream{ctx: context.Background()}

	err := <-forwardClientToServer(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "header failed")
}

func TestForwardClientToServer_SendHeaderError(t *testing.T) {
	src := &coverageClientStream{
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A}}},
		headerMD:  metadata.Pairs("x", "1"),
	}
	dst := &coverageServerStream{ctx: context.Background(), sendHeaderErr: errors.New("send header failed")}

	err := <-forwardClientToServer(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send header failed")
}

func TestForwardClientToServer_SendMsgError(t *testing.T) {
	src := &coverageClientStream{
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A}}, {Data: []byte{0x0B}}},
		headerMD:  metadata.Pairs("x", "1"),
	}
	dst := &coverageServerStream{ctx: context.Background(), sendErr: errors.New("send failed")}

	err := <-forwardClientToServer(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send failed")
}

func TestForwardServerToClient_SendMsgError(t *testing.T) {
	src := &coverageServerStream{ctx: context.Background(), recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A}}}}
	dst := &coverageClientStream{sendErr: errors.New("send failed")}

	err := <-forwardServerToClient(src, dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send failed")
}

func TestGrpcToGRPC_NoMethod(t *testing.T) {
	g := newCoverageHandler(time.Second)
	err := g.grpcToGRPC(nil, &coverageServerStream{ctx: context.Background()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "could not determine method from server stream")
}

func TestGrpcToGRPC_NoIncomingMetadata(t *testing.T) {
	g := newCoverageHandler(time.Second)
	ctx := streamContextWithMethod("/raw.Test/Echo")
	// context has a server transport stream but no incoming metadata.
	err := g.grpcToGRPC(nil, &coverageServerStream{ctx: ctx})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "could not get metadata")
}

func TestGrpcToGRPC_CannotDialService(t *testing.T) {
	// Grab a free address and close the listener so the connection is refused.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	refusedAddr := listener.Addr().String()
	require.NoError(t, listener.Close())

	conn, err := grpc.NewClient(refusedAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	g := newCoverageHandler(time.Second)
	g.grpcConn = conn

	ctx := metadata.NewIncomingContext(
		streamContextWithMethod("/raw.Test/Echo"),
		metadata.Pairs("x-test", "1"),
	)
	err = g.grpcToGRPC(nil, &coverageServerStream{ctx: ctx})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "can't connect to service")
}

// rawStreamEchoer is the interface used to register the raw streaming service.
type rawStreamEchoer interface {
	rawStreamHandler(grpc.ServerStream) error
}

type rawStreamService struct {
	handler func(any, grpc.ServerStream) error
}

func (s *rawStreamService) rawStreamHandler(stream grpc.ServerStream) error {
	return s.handler(nil, stream)
}

func newRawEchoDesc(handler func(any, grpc.ServerStream) error) *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: "raw.Test",
		HandlerType: (*rawStreamEchoer)(nil),
		Streams: []grpc.StreamDesc{{
			StreamName:    "Echo",
			Handler:       handler,
			ServerStreams: true,
			ClientStreams: true,
		}},
		Metadata: "raw.Test",
	}
}

func startRawServer(t *testing.T, handler func(any, grpc.ServerStream) error) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	server.RegisterService(newRawEchoDesc(handler), &rawStreamService{handler: handler})

	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func echoStreamHandler(_ any, stream grpc.ServerStream) error {
	frame := &codec.GrpcFrame{}
	for {
		if err := stream.RecvMsg(frame); err != nil {
			return err
		}
		if err := stream.SendMsg(&codec.GrpcFrame{Data: append([]byte("echo:"), frame.Data...)}); err != nil {
			return err
		}
	}
}

func TestGrpcToGRPC_FullRoundTrip(t *testing.T) {
	// Backend service that echoes messages.
	backendListener := startRawServer(t, echoStreamHandler)

	backendConn, err := grpc.NewClient(backendListener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = backendConn.Close() })

	// Front service proxying to the backend via grpcToGRPC.
	proxy := &grpcHandler{
		grpcConn: backendConn,
		enc:      "proto",
		timeout:  10 * time.Second,
	}
	frontListener := startRawServer(t, func(_ any, stream grpc.ServerStream) error {
		return proxy.grpcToGRPC(nil, stream)
	})

	clientConn, err := grpc.NewClient(frontListener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientConn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-test", "1"))

	stream, err := clientConn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}, "/raw.Test/Echo", grpc.CallContentSubtype("proto"))
	require.NoError(t, err)

	require.NoError(t, stream.SendMsg(&codec.GrpcFrame{Data: []byte("hello")}))
	require.NoError(t, stream.CloseSend())

	reply := &codec.GrpcFrame{}
	require.NoError(t, stream.RecvMsg(reply))
	assert.Equal(t, []byte("echo:hello"), reply.Data)
}
