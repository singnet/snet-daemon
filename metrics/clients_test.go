// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/health/grpc_health_v1"
)

// server is used to implement api.HeartbeatServer
type server struct{}

type ClientTestSuite struct {
	suite.Suite
	serviceURL string
	server     *grpc.Server
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.server.GracefulStop()
}
func (suite *ClientTestSuite) SetupSuite() {
	SetNoHeartbeatURLState(false)
	suite.serviceURL = "http://localhost:1111"
	suite.server = setAndServe()
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *server) Check(ctx context.Context, in *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}

const (
	testPort = ":33333"
)

type clientImplHeartBeat struct {
}

// Check implements `service Health`.
func (service *clientImplHeartBeat) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}

func (service *clientImplHeartBeat) Watch(*pb.HealthCheckRequest, pb.Health_WatchServer) error {
	return nil
}

// mocks grpc service endpoint for unit tests
func StartMockGrpcService() {
	ch := make(chan int)
	go func() {
		lis, err := net.Listen("tcp", testPort)
		if err != nil {
			panic(err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterHealthServer(grpcServer, &clientImplHeartBeat{})
		ch <- 0
		err = grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()
	_ = <-ch
}

func (suite *ClientTestSuite) Test_callgRPCServiceHeartbeat() {
	// Start the grpc mock server
	StartMockGrpcService()

	serviceURL := "localhost" + testPort
	heartbeat, err := callgRPCServiceHeartbeat(serviceURL)
	assert.Equal(suite.T(), nil, err)

	assert.NotEqual(suite.T(), `{}`, string(heartbeat), "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), pb.HealthCheckResponse_SERVING.String(), heartbeat.String())

	serviceURL = "localhost:26000"
	heartbeat, err = callgRPCServiceHeartbeat(serviceURL)
	assert.True(suite.T(), err != nil)
}

func (suite *ClientTestSuite) Test_callHTTPServiceHeartbeat() {
	serviceURL := suite.serviceURL + "/heartbeat"
	heartbeat, err := callHTTPServiceHeartbeat(serviceURL)
	assert.False(suite.T(), err != nil)
	assert.NotEqual(suite.T(), string(heartbeat), `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), string(heartbeat), `{"serviceID":"SERVICE001","status":"SERVING"}`,
		"Unexpected service heartbeat")

	heartbeat, err = callHTTPServiceHeartbeat(suite.serviceURL)
	assert.True(suite.T(), err != nil)
}

// TODO: refactor register service

// func (suite *ClientTestSuite) Test_callRegisterService() {

// 	daemonID := GetDaemonID()

// 	result := callRegisterService(daemonID, suite.serviceURL+"/register")
// 	assert.Equal(suite.T(), true, result)

// 	result = callRegisterService(daemonID, suite.serviceURL+"/registererror")
// 	assert.Equal(suite.T(), false, result)

// }
