//go:generate protoc grpc_test.proto --go-grpc_out=paths=source_relative:. --go_out=paths=source_relative:.

package handler

import (
	"context"
	"errors"
	"net"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

type GrpcTestSuite struct {
	suite.Suite
}

func (suite *GrpcTestSuite) SetupSuite() {
}

func (suite *GrpcTestSuite) TearDownSuite() {
}

func TestGrpcTestSuite(t *testing.T) {
	suite.Run(t, new(GrpcTestSuite))
}

type exampleServiceMock struct {
	output *Output
	err    error
}

func (service *exampleServiceMock) mustEmbedUnimplementedExampleServiceServer() {
	//TODO implement me
	panic("implement me")
}

func (service *exampleServiceMock) Ping(context context.Context, input *Input) (output *Output, err error) {
	return service.output, service.err
}

func startServiceAndClient(service ExampleServiceServer) (ExampleServiceClient, *grpc.ClientConn) {
	ch := make(chan int)
	go func() {
		listener, err := net.Listen("tcp", ":12345")
		if err != nil {
			panic(err)
		}

		server := grpc.NewServer()
		RegisterExampleServiceServer(server, service)

		ch <- 0

		server.Serve(listener)
	}()

	_ = <-ch

	connection, err := grpc.Dial("localhost:12345", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client := NewExampleServiceClient(connection)

	return client, connection
}

func (suite *GrpcTestSuite) TestReturnCustomErrorCodeViaGrpc() {
	expectedErr := status.Newf(1000, "error message").Err()
	client, connection := startServiceAndClient(&exampleServiceMock{err: expectedErr})
	defer connection.Close()

	_, err := client.Ping(context.Background(), &Input{Message: "ping"})

	assert.Equal(suite.T(), status.Code(err), status.Code(expectedErr))
	assert.Equal(suite.T(), status.Convert(err).Message(), status.Convert(expectedErr).Message())
}

func (suite *GrpcTestSuite) TestPassThroughEndPoint() {
	passthroughURL, err := url.Parse("http://localhost:8080")
	assert.Equal(suite.T(), passthroughURL.Scheme, "http")
	assert.Nil(suite.T(), err)
	passthroughURL, err = url.Parse("https://localhost:8080")
	assert.Equal(suite.T(), passthroughURL.Scheme, "https")
	passthroughURL, err = url.Parse("localhost:8080")
	assert.NotEqual(suite.T(), passthroughURL.Scheme, "https")
	passthroughURL, err = url.Parse("0.0.0.0:7000")
	assert.NotNil(suite.T(), err)
	passthroughURL, err = url.Parse("http://somedomain")
	assert.Equal(suite.T(), passthroughURL.Scheme, "http")
}

func TestHttpCredentials(t *testing.T) {
	var creds = []serviceCredentials{
		{serviceCredential{
			Key:      "api-key",
			Value:    "123abc",
			Location: "query",
		}},
		{serviceCredential{
			Key:      "X-Api-Key",
			Value:    "123abc",
			Location: "header",
		}},
	}

	for _, v := range creds {
		assert.Nil(t, v.validate())
	}

	var invalidCreds = []serviceCredentials{
		{serviceCredential{
			Key:      "api-key",
			Value:    "123abc",
			Location: "from",
		}},
		{serviceCredential{
			Key:      "X-Api-Key",
			Value:    "123abc",
			Location: "",
		}},
		{serviceCredential{
			Key:      "",
			Value:    "123abc",
			Location: "header",
		}},
	}

	for _, v := range invalidCreds {
		assert.NotNil(t, v.validate())
	}
}
