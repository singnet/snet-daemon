package handler

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestJsonToProtoRejectsInvalidResponse(t *testing.T) {
	for _, input := range []string{`not-json`, `{"bar":[]}`} {
		t.Run(input, func(t *testing.T) {
			_, err := jsonToProto(httpRoundTripDescriptors(t), []byte(input), "ExistingMethod")
			require.ErrorContains(t, err, "can't convert json to proto")
		})
	}
}

func TestBinaryMetadataRequiresExactlyOneValue(t *testing.T) {
	for _, md := range []metadata.MD{
		{}, metadata.Pairs("request-bin", "one", "request-bin", "two"),
	} {
		value, err := GetBytes(md, "request-bin")
		require.Nil(t, value)
		require.NotNil(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err.Err()))
	}
}

func TestProxyRejectsFirstFrameFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*grpcHandler, *coverageServerStream) error
	}{
		{"jsonrpc", func(g *grpcHandler, stream *coverageServerStream) error { return g.grpcToJSONRPC(nil, stream) }},
		{"process", func(g *grpcHandler, stream *coverageServerStream) error { return g.grpcToProcess(nil, stream) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newCoverageHandler(time.Second)
			stream := &coverageServerStream{ctx: streamContextWithMethod("/test.Service/run"), recvErr: errors.New("frame interrupted")}
			err := tc.call(g, stream)
			require.Equal(t, codes.Internal, status.Code(err))
			require.ErrorContains(t, err, "frame interrupted")
			require.Empty(t, stream.sent)
		})
	}
}
