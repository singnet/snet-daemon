package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bufbuild/protocompile/linker"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func httpRoundTripDescriptors(t *testing.T) linker.Files {
	t.Helper()
	return getDescriptors(t, map[string]string{
		"test.proto": `
			syntax = "proto3";
			package test;
			service TestService {
				rpc ExistingMethod (TestRequest) returns (TestResponse);
			}
			message TestRequest {
				string foo = 1;
			}
			message TestResponse {
				string bar = 1;
			}
		`,
	})
}

type failingRoundTripper struct{ err error }

func (f failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

func TestGrpcToHTTP_InvalidEndpoint(t *testing.T) {
	g := newCoverageHandler(time.Second)
	g.passthroughEndpoint = "://invalid"
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "can't parse service_endpoint")
}

func TestGrpcToHTTP_HttpClientError(t *testing.T) {
	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = "http://localhost:1"
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}
	g.httpClient = &http.Client{Transport: failingRoundTripper{err: errors.New("connection failed")}}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error executing HTTP service")
}

func TestGrpcToHTTP_UpstreamError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "bad gateway")
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}
	g.serviceCredentials = serviceCredentials{
		{Key: "api-key", Value: "secret", Location: query},
		{Key: "extra", Value: "value", Location: body},
	}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, err.Error(), "upstream http status 502")
}

func TestGrpcToHTTP_SendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, `{"bar":"1"}`)
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	g.serviceMetaData = &blockchain.ServiceMetadata{ProtoDescriptors: httpRoundTripDescriptors(t)}

	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{
		ctx:       ctx,
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}},
		sendErr:   errors.New("send failed"),
	}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error sending response from HTTP service")
}

func TestGrpcToHTTP_NoMethod(t *testing.T) {
	g := newCoverageHandler(time.Second)
	err := g.grpcToHTTP(nil, &coverageServerStream{ctx: context.Background()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "could not determine method from server stream")
}

func TestGrpcToHTTP_InvalidMethodFormat(t *testing.T) {
	g := newCoverageHandler(time.Second)
	for _, method := range []string{"/only-service", "/svc/", "/svc/m1/m2"} {
		ctx := metadata.NewIncomingContext(streamContextWithMethod(method), metadata.Pairs("x", "1"))
		err := g.grpcToHTTP(nil, &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte("{}")}}})
		require.Error(t, err, "method: %s", method)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Contains(t, err.Error(), "unexpected grpc method format")
	}
}

func TestGrpcToHTTP_RecvError(t *testing.T) {
	g := newCoverageHandler(time.Second)
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvErr: errors.New("recv failed")}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "error receiving grpc msg")
}

func TestGrpcToHTTP_ProtoToJsonError(t *testing.T) {
	g := newCoverageHandler(time.Second)
	g.serviceMetaData = newCoverageMetadata("http", "proto")
	ctx := metadata.NewIncomingContext(streamContextWithMethod("/test.TestService/ExistingMethod"), metadata.Pairs("x", "1"))
	stream := &coverageServerStream{ctx: ctx, recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}}}

	err := g.grpcToHTTP(nil, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "protoToJson error")
}

func TestGrpcToHTTP_FullRoundTrip(t *testing.T) {
	descriptors := httpRoundTripDescriptors(t)

	var gotQuery, gotHeader string
	var gotBody []byte

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("api-key")
		gotHeader = r.Header.Get("X-Api-Key")
		var err error
		gotBody, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, `{"bar":"1"}`)
	}))
	defer backend.Close()

	g := newCoverageHandler(5 * time.Second)
	g.passthroughEndpoint = backend.URL
	g.serviceMetaData = &blockchain.ServiceMetadata{
		ServiceType:      "http",
		ProtoDescriptors: descriptors,
	}
	g.serviceCredentials = serviceCredentials{
		{Key: "api-key", Value: "secret", Location: query},
		{Key: "X-Api-Key", Value: "secret2", Location: header},
		{Key: "extra", Value: "value", Location: body},
	}

	ctx := metadata.NewIncomingContext(
		streamContextWithMethod("/test.TestService/ExistingMethod"),
		metadata.Pairs("x", "1"),
	)
	stream := &coverageServerStream{
		ctx:       ctx,
		recvQueue: []*codec.GrpcFrame{{Data: []byte{0x0A, 0x01, 'x'}}},
	}

	require.NoError(t, g.grpcToHTTP(nil, stream))

	assert.Equal(t, "secret", gotQuery)
	assert.Equal(t, "secret2", gotHeader)
	assert.Contains(t, string(gotBody), `"extra":"value"`)
	assert.Contains(t, string(gotBody), `"foo":"x"`)

	require.Len(t, stream.sent, 1)
	assert.NotNil(t, stream.sent[0])
}
