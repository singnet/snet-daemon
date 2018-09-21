package test

import (
	"context"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/resources/test/echoservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func testGRPC(t *testing.T, h harness, port string) {
	job := h.createSignedJob()
	conn, err := grpc.Dial("127.0.0.1:"+port, grpc.WithInsecure())
	require.NoError(t, err, "Unable to dial gRPC client")
	defer conn.Close()

	grpcClient := echoservice.NewEchoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(
		ctx,
		blockchain.JobAddressHeader, job.address,
		blockchain.JobSignatureHeader, job.signature,
	)
	r, err := grpcClient.Echo(ctx, &echoservice.EchoEnvelope{
		Message: "from gRPC",
	})
	assert.NoError(t, err, "got error performing gRPC echo request")
	assert.NotEmpty(t, r, "expected response body on successful echo gRPC request")
	assert.Equal(t, r.GetMessage(), "from gRPC")

	// Failure paths
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err = grpcClient.Echo(ctx, &echoservice.EchoEnvelope{
		Message: "this request is missing headers",
	})
	assert.Error(t, err, "expected error without job headers")
	assert.Empty(t, r)
}
