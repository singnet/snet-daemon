//go:generate protoc grpc_stream_test.proto --go-grpc_out=. --go_out=.

package intergrationtests

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type integrationTestEnvironment struct {
	serverA   *grpc.Server
	listenerA *net.Listener
	serverB   *grpc.Server
	listenerB *net.Listener
}

func setupTestConfig() {
	testConfigJson := `
{
	"blockchain_enabled": true,
	"blockchain_network_selected": "sepolia",
	"daemon_endpoint": "127.0.0.1:8080",
	"daemon_group_name":"default_group",
	"payment_channel_storage_type": "etcd",
	"ipfs_endpoint": "http://ipfs.singularitynet.io:80", 
	"ipfs_timeout" : 30,
	"passthrough_enabled": true,
	"service_endpoint":"http://127.0.0.1:5002",
	"service_id": "ExampleServiceId", 
	"organization_id": "ExampleOrganizationId",
	"metering_enabled": false,
	"ssl_cert": "",
	"ssl_key": "",
	"max_message_size_in_mb" : 4,
	"daemon_type": "grpc",
    "enable_dynamic_pricing":false,
	"allowed_user_flag" :false,
	"auto_ssl_domain": "",
	"auto_ssl_cache_dir": ".certs",
	"private_key": "",
	"log":  {
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "text",
			"timestamp_format": "2006-01-02T15:04:05.999Z07:00"
		},
		"output": {
			"type": ["file", "stdout"],
			"file_pattern": "./snet-daemon.%Y%m%d.log",
			"current_link": "./snet-daemon.log",
			"max_size_in_mb": 10,
			"max_age_in_days": 7,
			"rotation_count": 0
		},
		"hooks": []
	},
	"payment_channel_storage_client": {
		"connection_timeout": "0s",
		"request_timeout": "0s",
		"hot_reload": true
    },
	"payment_channel_storage_server": {
		"id": "storage-1",
		"scheme": "http",
		"host" : "127.0.0.1",
		"client_port": 2379,
		"peer_port": 2380,
		"token": "unique-token",
		"cluster": "storage-1=http://127.0.0.1:2380",
		"startup_timeout": "1m",
		"data_dir": "storage-data-dir-1.etcd",
		"log_level": "info",
		"log_outputs": ["./etcd-server.log"],
		"enabled": false
	},
	"alerts_email": "", 
	"service_heartbeat_type": "http",
    "token_expiry_in_minutes": 1440,
    "model_training_enabled": false
}`

	var testConfig = viper.New()
	err := config.ReadConfigFromJsonString(testConfig, testConfigJson)
	if err != nil {
		zap.L().Fatal("Error in reading config")
	}

	config.SetVip(testConfig)
}

func startServerA(port string, h grpc.StreamHandler) (*grpc.Server, *net.Listener) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		zap.L().Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	RegisterExampleStreamingServiceServer(grpcServer, &ServiceA{h: h})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			zap.L().Fatal("Failed to serve", zap.Error(err))
		}
	}()
	return grpcServer, &lis
}

func startServerB(port string) (*grpc.Server, *net.Listener) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		zap.L().Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	RegisterExampleStreamingServiceServer(grpcServer, &ServiceB{})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			zap.L().Fatal("Failed to serve", zap.Error(err))
		}
	}()
	return grpcServer, &lis
}

func setupEnvironment() *integrationTestEnvironment {
	setupTestConfig()
	serviceMetadata := &blockchain.ServiceMetadata{
		ServiceType: "grpc",
		Encoding:    "proto",
	}
	grpcToGrpc := handler.NewGrpcHandler(serviceMetadata)
	grpcServerA, listenerA := startServerA(":5001", grpcToGrpc)
	grpcServerB, listenerB := startServerB(":5002")

	testEnv := &integrationTestEnvironment{
		serverA:   grpcServerA,
		listenerA: listenerA,
		serverB:   grpcServerB,
		listenerB: listenerB,
	}

	return testEnv
}

func teardownEnvironment(env *integrationTestEnvironment) {
	env.serverA.Stop()
	env.serverB.Stop()
	(*env.listenerA).Close()
	(*env.listenerB).Close()
}

type ServiceA struct {
	UnimplementedExampleStreamingServiceServer
	h grpc.StreamHandler
}

type ServiceB struct {
	UnimplementedExampleStreamingServiceServer
}

func (s *ServiceA) Stream(stream ExampleStreamingService_StreamServer) error {
	// Forward the stream to grpcToGrpc handler
	err := s.h(nil, stream)
	if err != nil {
		return err
	}
	return nil
}

func (s *ServiceB) Stream(stream ExampleStreamingService_StreamServer) error {
	for {
		// Receive the input from the proxied stream
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Send back the response to the stream
		err = stream.Send(&OutStream{
			Message: "Response from Server B: " + req.Message,
		})
		if err != nil {
			return err
		}
	}
}

func TestGrpcToGrpc_StreamingIntegration(t *testing.T) {
	// Setup test environment with real servers
	testEnv := setupEnvironment()
	defer teardownEnvironment(testEnv)

	// Create a gRPC client to interact with Server A
	connA, err := grpc.NewClient((*testEnv.listenerA).Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer connA.Close()

	clientA := NewExampleStreamingServiceClient(connA)

	// Simulate a bidirectional streaming RPC call from clientA to Server A
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := clientA.Stream(ctx)
	require.NoError(t, err)

	// Send a message from the client A to Server A
	err = stream.Send(&InStream{Message: "Hello from client A"})
	require.NoError(t, err)

	// Receive the response from Server B (proxied by grpcToGrpc)
	resp, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "Response from Server B: Hello from client A", resp.Message)

	// Close the stream
	stream.CloseSend()
}
