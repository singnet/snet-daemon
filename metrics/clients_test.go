// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"testing"
)

// server is used to implement api.HeartbeatServer
type server struct{}

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

func Test_callgRPCServiceHeartbeat(t *testing.T) {
	// Start the grpc mock server
	StartMockGrpcService()

	serviceURL := "localhost" + testPort
	heartbeat, err := callgRPCServiceHeartbeat(serviceURL)
	assert.False(t, err != nil)

	assert.NotEqual(t, `{}`, string(heartbeat), "Service Heartbeat must not be empty.")
	assert.Equal(t, heartbeat.String(), pb.HealthCheckResponse_SERVING.String())

	serviceURL = "localhost:26000"
	heartbeat, err = callgRPCServiceHeartbeat(serviceURL)
	assert.True(t, err != nil)
}

func Test_callHTTPServiceHeartbeat(t *testing.T) {
	serviceURL := "http://demo5343751.mockable.io/heartbeat"
	heartbeat, err := callHTTPServiceHeartbeat(serviceURL)
	assert.False(t, err != nil)
	assert.NotEqual(t, string(heartbeat), `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(t, string(heartbeat), `{"serviceID":"SERVICE001", "status":"SERVING"}`,
		"Unexpected service heartbeat")

	/*	var sHeartbeat pb.HeartbeatMsg
		err = json.Unmarshal(heartbeat, &sHeartbeat)
		assert.True(t, err != nil)
		assert.Equal(t, sHeartbeat.ServiceID, "SERVICE001", "Unexpected service ID")

	*/serviceURL = "http://demo8325345.mockable.io"
	heartbeat, err = callHTTPServiceHeartbeat(serviceURL)
	assert.True(t, err != nil)
}

func Test_callRegisterService(t *testing.T) {
	serviceURL := "https://demo5343751.mockable.io/register"
	daemonID := GetDaemonID()

	result := callRegisterService(daemonID, serviceURL)
	assert.Equal(t, true, result)

	serviceURL = "https://demo5343751.mockable.io/registererror"
	result = callRegisterService(daemonID, serviceURL)
	assert.Equal(t, false, result)

}
