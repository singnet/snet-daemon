// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	suite.serviceURL = "http://localhost:1111"
	suite.server = setAndServe()
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *server) Check(ctx context.Context, in *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}

type clientImplHeartBeat struct {
}

func (service *clientImplHeartBeat) List(ctx context.Context, request *pb.HealthListRequest) (*pb.HealthListResponse, error) {
	//TODO implement me
	panic("implement me")
}

// Check implements `service Health`.
func (service *clientImplHeartBeat) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}

func (service *clientImplHeartBeat) Watch(*pb.HealthCheckRequest, pb.Health_WatchServer) error {
	return nil
}

// mocks grpc service endpoint for unit tests
func StartMockGrpcService(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := grpc.NewServer()
	pb.RegisterHealthServer(grpcServer, &clientImplHeartBeat{})
	done := make(chan error, 1)
	go func() {
		done <- grpcServer.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		if err := <-done; !errors.Is(err, grpc.ErrServerStopped) {
			assert.NoError(t, err)
		}
	})
	return lis.Addr().String()
}

func (suite *ClientTestSuite) Test_callgRPCServiceHeartbeat() {
	// Start the grpc mock server
	serviceURL := StartMockGrpcService(suite.T())
	heartbeat, err := callGrpcServiceHeartbeat(serviceURL)
	assert.NoError(suite.T(), err)

	assert.NotEqual(suite.T(), `{}`, string(heartbeat), "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), pb.HealthCheckResponse_SERVING.String(), heartbeat.String())

	serviceURL = "localhost:26000"
	heartbeat, err = callGrpcServiceHeartbeat(serviceURL)
	assert.Error(suite.T(), err)
}

func (suite *ClientTestSuite) Test_callHTTPServiceHeartbeat() {
	serviceURL := suite.serviceURL + "/heartbeat"
	heartbeat, err := callHTTPServiceHeartbeat(serviceURL)
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), string(heartbeat), `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), string(heartbeat), `{"serviceID":"SERVICE001","status":"SERVING"}`,
		"Unexpected service heartbeat")

	heartbeat, err = callHTTPServiceHeartbeat(suite.serviceURL)
	assert.Error(suite.T(), err)
}

// TODO: refactor register service

// func (suite *ClientTestSuite) Test_callRegisterService() {

// 	daemonID := GetDaemonID()

// 	result := callRegisterService(daemonID, suite.serviceURL+"/register")
// 	assert.Equal(suite.T(), true, result)

// 	result = callRegisterService(daemonID, suite.serviceURL+"/registererror")
// 	assert.Equal(suite.T(), false, result)

// }
