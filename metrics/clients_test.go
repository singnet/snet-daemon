// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"testing"
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
		grpcServer.Serve(lis)
	}()
	_ = <-ch
}

func (suite *ClientTestSuite) Test_callgRPCServiceHeartbeat() {
	// Start the grpc mock server
	StartMockGrpcService()

	serviceURL := "localhost" + testPort
	heartbeat, err := callgRPCServiceHeartbeat(serviceURL)
	assert.False(suite.T(), err != nil)

	assert.NotEqual(suite.T(), `{}`, string(heartbeat), "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), heartbeat.String(), pb.HealthCheckResponse_SERVING.String())

	serviceURL = "localhost:26000"
	heartbeat, err = callgRPCServiceHeartbeat(serviceURL)
	assert.True(suite.T(), err != nil)
}

func (suite *ClientTestSuite) Test_callHTTPServiceHeartbeat() {
	serviceURL := "http://localhost:1111/heartbeat"
	heartbeat, err := callHTTPServiceHeartbeat(serviceURL)
	assert.False(suite.T(), err != nil)
	assert.NotEqual(suite.T(), string(heartbeat), `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), string(heartbeat), `{"serviceID":"SERVICE001","status":"SERVING"}`,
		"Unexpected service heartbeat")

	/*	var sHeartbeat pb.HeartbeatMsg
		err = json.Unmarshal(heartbeat, &sHeartbeat)
		assert.True(t, err != nil)
		assert.Equal(t, sHeartbeat.ServiceID, "SERVICE001", "Unexpected service ID")

	*/serviceURL = "http://localhost:1111"
	heartbeat, err = callHTTPServiceHeartbeat(serviceURL)
	assert.True(suite.T(), err != nil)
}

func (suite *ClientTestSuite) Test_callRegisterService() {
	serviceURL := "http://localhost:1111/register"
	daemonID := GetDaemonID()

	result := callRegisterService(daemonID, serviceURL)
	assert.Equal(suite.T(), true, result)

	serviceURL = "http://localhost:1111/registererror"
	result = callRegisterService(daemonID, serviceURL)
	assert.Equal(suite.T(), false, result)

}
