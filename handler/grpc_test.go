//go:generate protoc grpc_test.proto --go-grpc_out=. --go_out=.

package handler

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

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

	assert.Equal(suite.T(), err, expectedErr)
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
