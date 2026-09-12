package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/bufbuild/protocompile/linker"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	// Allow the test binary to be re-executed as a sub-process for grpcToProcess tests.
	if os.Getenv("SNET_PASSTHROUGH_HELPER") != "1" {
		return
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(data); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func setConfigForTest(t *testing.T, key string, value any) {
	t.Helper()
	previous := config.Vip().Get(key)
	t.Cleanup(func() { config.Vip().Set(key, previous) })
	config.Vip().Set(key, value)
}

type coverageTransportStream struct {
	method string
}

func (s coverageTransportStream) Method() string               { return s.method }
func (s coverageTransportStream) SetHeader(metadata.MD) error  { return nil }
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

func (s *coverageServerStream) Context() context.Context    { return s.ctx }
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
func (s *coverageClientStream) Trailer() metadata.MD     { return s.trailersMD }
func (s *coverageClientStream) CloseSend() error         { return s.closeSend }

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

func TestGrpcHandler_GrpcConn(t *testing.T) {
	g := &grpcHandler{
		grpcConn:      &grpc.ClientConn{},
		grpcModelConn: &grpc.ClientConn{},
	}

	assert.Same(t, g.grpcModelConn, g.GrpcConn(true))
	assert.Same(t, g.grpcConn, g.GrpcConn(false))
}

func TestNewGrpcHandler_PassthroughDisabledReturnsLoopback(t *testing.T) {
	setConfigForTest(t, config.PassthroughEnabledKey, false)

	got := NewGrpcHandler(newCoverageMetadata("grpc", "proto"))

	require.NotNil(t, got)
	assert.Equal(t, reflect.ValueOf(grpcLoopback).Pointer(), reflect.ValueOf(got).Pointer())
}

func TestNewGrpcHandler_ServiceTypes(t *testing.T) {
	setConfigForTest(t, config.PassthroughEnabledKey, true)

	cases := []string{"grpc", "jsonrpc", "http", "process"}
	for _, serviceType := range cases {
		t.Run(serviceType, func(t *testing.T) {
			handler := NewGrpcHandler(newCoverageMetadata(serviceType, "proto"))
			require.NotNil(t, handler)

			// All passthrough handlers start by resolving the method from the
			// stream and fail when it can't be determined. grpcLoopback would
			// succeed here, which proves the switch returned a passthrough handler.
			err := handler(nil, &coverageServerStream{ctx: context.Background()})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "could not determine method from server stream")
		})
	}
}

func TestNewGrpcHandler_UnknownServiceTypeReturnsNil(t *testing.T) {
	setConfigForTest(t, config.PassthroughEnabledKey, true)

	assert.Nil(t, NewGrpcHandler(newCoverageMetadata("unsupported", "proto")))
}

func TestGetConnection_InsecureWithScheme(t *testing.T) {
	g := newCoverageHandler(time.Second)
	conn := g.getConnection("http://localhost:12345")
	require.NotNil(t, conn)
	assert.Equal(t, "localhost:12345", conn.Target())
	require.NoError(t, conn.Close())
}

func TestGetConnection_InsecureWithoutScheme(t *testing.T) {
	g := newCoverageHandler(time.Second)
	conn := g.getConnection("localhost:12346")
	require.NotNil(t, conn)
	assert.Equal(t, "localhost:12346", conn.Target())
	require.NoError(t, conn.Close())
}

func TestGetConnection_Secure(t *testing.T) {
	g := newCoverageHandler(time.Second)
	conn := g.getConnection("https://localhost:12347")
	require.NotNil(t, conn)
	assert.Equal(t, "localhost:12347", conn.Target())
	require.NoError(t, conn.Close())
}

func TestGrpcLoopback(t *testing.T) {
	input := &codec.GrpcFrame{Data: []byte("hello")}
	stream := &coverageServerStream{
		ctx:       context.Background(),
		recvQueue: []*codec.GrpcFrame{input},
	}

	require.NoError(t, grpcLoopback(nil, stream))
	require.Len(t, stream.sent, 1)
	assert.Equal(t, []byte("hello"), stream.sent[0].(*codec.GrpcFrame).Data)
}

func TestGrpcLoopback_RecvError(t *testing.T) {
	stream := &coverageServerStream{
		ctx:     context.Background(),
		recvErr: errors.New("recv failed"),
	}

	err := grpcLoopback(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error receiving request")
}

func TestGrpcLoopback_SendError(t *testing.T) {
	stream := &coverageServerStream{
		ctx:       context.Background(),
		recvQueue: []*codec.GrpcFrame{{Data: []byte("ping")}},
		sendErr:   errors.New("send failed"),
	}

	err := grpcLoopback(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error sending response")
}

func httpRoundTripDescriptors(t *testing.T) linker.Files {
	t.Helper()
	return getDescriptors(t, map[string]string{
		"test.proto": `
			syntax = "proto3";
			package test;
			service TestService {
				rpc ExistingMethod (TestRequest) returns (TestResponse);
			}
			message TestRequest {
				string foo = 1;
			}
			message TestResponse {
				string bar = 1;
			}
		`,
	})
}

type failingRoundTripper struct{ err error }

func (f failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

func TestGrpcToHTTP_InvalidEndpoint(t *testing.T) {
	g := newCoverageHandler(time.Second)
	g.passthroughEndpoint = "://invalid"
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "can't parse service_endpoint")
}

func TestGrpcToHTTP_HttpClientError(t *testing.T) {
	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = "http://localhost:1"
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}
	g.httpClient = &http.Client{Transport: failingRoundTripper{err: errors.New("connection failed")}}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error executing HTTP service")
}

func TestGrpcToHTTP_UpstreamError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "bad gateway")
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}
	g.serviceCredentials = serviceCredentials{
		{Key: "api-key", Value: "secret", Location: query},
		{Key: "extra", Value: "value", Location: body},
	}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, err.Error(), "upstream http status 502")
}

func TestGrpcToHTTP_SendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, `{"bar":"1"}`)
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{
		ctx:       ctx,
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}},
		sendErr:   errors.New("send failed"),
	}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error sending response from HTTP service")
}

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

func TestGrpcToHTTP_NoMethod(t *testing.T) {
	g := newCoverageHandler(time.Second)
	err := g.grpcToHTTP(nil, &coverageServerStream{ctx: context.Background()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "could not determine method from server stream")
}

func TestGrpcToHTTP_InvalidMethodFormat(t *testing.T) {
	g := newCoverageHandler(time.Second)
	for _, method := range []string{"/only-service", "/svc/", "/svc/m1/m2"} {
		ctx := metadata.NewIncomingContext(streamContextWithMethod(method), metadata.Pairs("x", "1"))
		err := g.grpcToHTTP(nil, &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte("{}")}}})
		require.Error(t, err, "method: %s", method)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Contains(t, err.Error(), "unexpected grpc method format")
	}
}

func TestGrpcToHTTP_RecvError(t *testing.T) {
	g := newCoverageHandler(time.Second)
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvErr: errors.New("recv failed")}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error receiving grpc msg")
}

func TestGrpcToHTTP_ProtoToJsonError(t *testing.T) {
	g := newCoverageHandler(time.Second)
	g.serviceMetaData = newCoverageMetadata("http", "proto")
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "protoToJson error")
}

func TestGrpcToHTTP_FullRoundTrip(t *testing.T) {
	descriptors := httpRoundTripDescriptors(t)

	var gotQuery, gotHeader string
	var gotBody []byte

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("api-key")
		gotHeader = r.Header.Get("X-Api-Key")
		var err error
		gotBody, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, `{"bar":"1"}`)
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	g.serviceMetaData = &blockchain.ServiceMetadata{
		ServiceType:      "http",
		ProtoDescriptors: descriptors,
	}
	g.serviceCredentials = serviceCredentials{
		{Key: "api-key", Value: "secret", Location: query},
		{Key: "X-Api-Key", Value: "secret2", Location: header},
		{Key: "extra", Value: "value", Location: body},
	}

	ctx := metadata.NewIncomingContext(
		streamContextWithMethod("/test.TestService/ExistingMethod"),
		metadata.Pairs("x", "1"),
	)
	stream := &coverageServerStream{
		ctx:       ctx,
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}},
	}

	require.NoError(t, g.grpcToHTTP(nil, stream))

	assert.Equal(t, "secret", gotQuery)
	assert.Equal(t, "secret2", gotHeader)
	assert.Contains(t, string(gotBody), `"extra":"value"`)
	assert.Contains(t, string(gotBody), `"foo":"x"`)

	require.Len(t, stream.sent, 1)
	assert.NotNil(t, stream.sent[0])
}

func TestGrpcToJSONRPC_NoMethod(t *testing.T) {
	g := newCoverageHandler(time.Second)
	err := g.grpcToJSONRPC(nil, &coverageServerStream{ctx: context.Background()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "could not determine method from server stream")
}

func TestGrpcToJSONRPC_RecvError(t *testing.T) {
	g := newCoverageHandler(time.Second)
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/add"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvErr: errors.New("recv failed")}

	err := g.grpcToJSONRPC(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error receiving request")
}

func TestGrpcToJSONRPC_InvalidRequestBody(t *testing.T) {
	g := newCoverageHandler(time.Second)
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/add"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte("not-json")}}}

	err := g.grpcToJSONRPC(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error unmarshaling request")
}

func TestGrpcToJSONRPC_HttpClientError(t *testing.T) {
	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = "http://localhost:1"
	g.httpClient = &http.Client{Transport: failingRoundTripper{err: errors.New("connection failed")}}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/add"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte(`{}`)}}}

	err := g.grpcToJSONRPC(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error executing http call")
}

func TestGrpcToJSONRPC_UpstreamError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/add"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte(`{}`)}}}

	err := g.grpcToJSONRPC(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, err.Error(), "upstream http status 500")
}

func TestGrpcToJSONRPC_InvalidUpstreamResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, "not-a-json-rpc-response")
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/add"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte(`{}`)}}}

	err := g.grpcToJSONRPC(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "json-rpc error")
}

func TestGrpcToJSONRPC_FullRoundTrip(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, `{"result":{"message":"pong"},"error":null}`)
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/add"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte(`{"a":1}`)}}}

	require.NoError(t, g.grpcToJSONRPC(nil, stream))

	require.Len(t, stream.sent, 1)
	frame, ok := stream.sent[0].(*codec.GrpcFrame)
	require.True(t, ok)
	assert.JSONEq(t, `{"message":"pong"}`, string(frame.Data))
}

func TestGrpcToProcess_NoMethod(t *testing.T) {
	g := newCoverageHandler(time.Second)
	err := g.grpcToProcess(nil, &coverageServerStream{ctx: context.Background()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "could not determine method from server stream")
}

func TestGrpcToProcess_RecvError(t *testing.T) {
	g := newCoverageHandler(time.Second)
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/run"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvErr: errors.New("recv failed")}

	err := g.grpcToProcess(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error receiving request")
}

func TestGrpcToProcess_ProcessFailed(t *testing.T) {
	g := newCoverageHandler(time.Second)
	g.executable = "nonexistent-command-for-test-12345"
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/run"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte("input")}}}

	err := g.grpcToProcess(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "process failed")
}

func TestGrpcToProcess_FullRoundTrip(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)

	g := newCoverageHandler(10 * time.Second)
	g.executable = executable

	t.Setenv("SNET_PASSTHROUGH_HELPER", "1")

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/run"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte("hello from stdin")}}}

	require.NoError(t, g.grpcToProcess(nil, stream))

	require.Len(t, stream.sent, 1)
	frame, ok := stream.sent[0].(*codec.GrpcFrame)
	require.True(t, ok)
	assert.Equal(t, "hello from stdin", string(frame.Data))
}

func TestGrpcToProcess_SendError(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)

	g := newCoverageHandler(10 * time.Second)
	g.executable = executable
	g.serviceMetaData = newCoverageMetadata("process", "proto")

	t.Setenv("SNET_PASSTHROUGH_HELPER", "1")

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/run"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{
		ctx:       ctx,
		recvQueue: []*codec.GrpcFrame{{Data: []byte("data")}},
		sendErr:   errors.New("send failed"),
	}

	err = g.grpcToProcess(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error sending response")
}
