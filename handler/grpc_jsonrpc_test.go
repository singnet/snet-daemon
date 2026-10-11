package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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
