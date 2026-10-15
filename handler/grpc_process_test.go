package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	// Allow the test binary to be re-executed as a sub-process for grpcToProcess tests.
	mode := os.Getenv("SNET_PASSTHROUGH_HELPER")
	if mode != "1" && mode != "blocked" {
		return
	}
	if mode == "blocked" {
		<-time.After(time.Minute)
		os.Exit(0)
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

func TestGrpcToProcessCancellationWithLargeInput(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("SNET_PASSTHROUGH_HELPER", "blocked")
	g := newCoverageHandler(50 * time.Millisecond)
	g.executable = executable
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.Service/run"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: bytes.Repeat([]byte("x"), 1024*1024)}}}
	done := make(chan error, 1)
	go func() { done <- g.grpcToProcess(nil, stream) }()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("process cancellation blocked writing stdin")
	}
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
