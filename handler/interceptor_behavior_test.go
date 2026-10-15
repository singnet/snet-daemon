package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/configuration_service"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRateLimitInterceptorDecisions(t *testing.T) {
	expected := errors.New("provider failed")
	for _, tc := range []struct {
		name        string
		processing  int
		limit       rate.Limit
		burst       int
		handlerErr  error
		wantCode    codes.Code
		wantHandler bool
	}{
		{"processing stopped", configuration_service.StopProcessingAnyRequest, rate.Inf, 1, nil, codes.Unavailable, false},
		{"limit exhausted", configuration_service.StartProcessingAnyRequest, 0, 0, nil, codes.ResourceExhausted, false},
		{"accepted", configuration_service.StartProcessingAnyRequest, rate.Inf, 1, nil, codes.OK, true},
		{"provider error", configuration_service.StartProcessingAnyRequest, rate.Inf, 1, expected, codes.Unknown, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			interceptor := &rateLimitInterceptor{processRequest: tc.processing}
			interceptor.rateLimiter.SetLimit(tc.limit)
			interceptor.rateLimiter.SetBurst(tc.burst)
			called := false
			err := interceptor.intercept(nil, nil, nil, func(any, grpc.ServerStream) error { called = true; return tc.handlerErr })
			require.Equal(t, tc.wantCode, status.Code(err))
			require.Equal(t, tc.wantHandler, called)
			if tc.handlerErr != nil {
				require.ErrorIs(t, err, tc.handlerErr)
			}
		})
	}
}

func TestGrpcRateLimitInterceptorUsesConfiguredLimit(t *testing.T) {
	setConfigForTest(t, config.RateLimitPerMinute, "1")
	setConfigForTest(t, config.BurstSize, 1)
	interceptor := GrpcRateLimitInterceptor(&configuration_service.MessageBroadcaster{})
	calls := 0
	handler := func(any, grpc.ServerStream) error { calls++; return nil }
	require.NoError(t, interceptor(nil, nil, nil, handler))
	require.Equal(t, codes.ResourceExhausted, status.Code(interceptor(nil, nil, nil, handler)))
	require.Equal(t, 1, calls)
}

func TestMeteringInterceptorPublishesResponse(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		withMetadata, withBlock bool
		handlerErr, blockErr    error
		wantStatus, wantCode    string
	}{
		{"success", true, true, nil, nil, "success", "OK"},
		{"provider and block failure", true, true, status.Error(codes.Unavailable, "provider unavailable"), errors.New("chain offline"), "failed", "Unavailable"},
		{"without metadata or block reader", false, false, nil, nil, "success", "OK"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type capturedResponse struct {
				stats metrics.ResponseStats
				err   error
				path  string
			}
			received := make(chan capturedResponse, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var stats metrics.ResponseStats
				err := json.NewDecoder(r.Body).Decode(&stats)
				received <- capturedResponse{stats: stats, err: err, path: r.URL.Path}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			setConfigForTest(t, config.MeteringEndpoint, server.URL)
			// Invalid credentials bypass signing, allowing missing/failed block readers.
			setConfigForTest(t, config.PvtKeyForMetering, "invalid")
			ctx := streamContextWithMethod("/example.Service/Ping")
			if tc.withMetadata {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(ClientTypeHeader, "sdk", PaymentTypeHeader, "escrow", PaymentChannelIDHeader, "42"))
			}
			blocks := make(chan struct{}, 1)
			var currentBlock func() (*big.Int, error)
			if tc.withBlock {
				currentBlock = func() (*big.Int, error) { blocks <- struct{}{}; return big.NewInt(100), tc.blockErr }
			}
			called := false
			err := GrpcMeteringInterceptor(currentBlock)(nil, &coverageServerStream{ctx: ctx}, nil, func(any, grpc.ServerStream) error { called = true; return tc.handlerErr })
			require.True(t, called)
			require.ErrorIs(t, err, tc.handlerErr)
			select {
			case result := <-received:
				require.NoError(t, result.err)
				require.Equal(t, "/metering/usage", result.path)
				require.Equal(t, "/example.Service/Ping", result.stats.ServiceMethod)
				require.Equal(t, tc.wantStatus, result.stats.Status)
				require.Equal(t, tc.wantCode, result.stats.ResponseCode)
				if tc.withMetadata {
					require.Equal(t, "sdk", result.stats.ClientType)
					require.Equal(t, "42", result.stats.ChannelId)
					require.Equal(t, "escrow", result.stats.PaymentMode)
				}
				if tc.handlerErr != nil {
					require.Equal(t, tc.handlerErr.Error(), result.stats.ErrorMessage)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("metering did not publish the response")
			}
			if tc.withBlock {
				require.Len(t, blocks, 1)
			} else {
				require.Empty(t, blocks)
			}
		})
	}
}

type senderStreamPaymentHandler struct{ *unaryPaymentHandlerMock }

func (h senderStreamPaymentHandler) Payment(*GrpcStreamContext) (Payment, *GrpcError) {
	return h.payment, h.paymentErr
}

func TestStreamPaymentInterceptorPropagatesSenderWithoutMutatingMetadata(t *testing.T) {
	sender := common.HexToAddress("0x00000000000000000000000000000000000000ab")
	md := metadata.Pairs(PaymentTypeHeader, "sender", SnetUserAddressHeader, "original")
	stream := &coverageServerStream{ctx: metadata.NewIncomingContext(context.Background(), md), recvQueue: []*codec.GrpcFrame{{}}}
	paymentHandler := senderStreamPaymentHandler{&unaryPaymentHandlerMock{typ: "sender", payment: unaryPaymentMock{sender: sender}}}
	interceptor := GrpcPaymentValidationInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
	err := interceptor(nil, stream, &grpc.StreamServerInfo{}, func(_ any, wrapped grpc.ServerStream) error {
		forwarded, ok := metadata.FromIncomingContext(wrapped.Context())
		require.True(t, ok)
		require.Equal(t, []string{sender.Hex()}, forwarded.Get(SnetUserAddressHeader))
		require.Equal(t, []string{"streamIntercept"}, forwarded.Get("snet-daemon-debug"))
		return nil
	})
	require.NoError(t, err)
	require.True(t, paymentHandler.completeCalled)
	require.Equal(t, []string{"original"}, md.Get(SnetUserAddressHeader))
}

func TestStreamPaymentInterceptorRejectsInvalidRequestsBeforeHandler(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		recvErr error
		code    codes.Code
	}{
		{"missing metadata", context.Background(), nil, codes.InvalidArgument},
		{"first frame failure", metadata.NewIncomingContext(context.Background(), metadata.MD{}), io.ErrUnexpectedEOF, codes.Unknown},
		{"unknown payment type", metadata.NewIncomingContext(context.Background(), metadata.Pairs(PaymentTypeHeader, "unknown")), nil, codes.InvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			paymentHandler := &paymentHandlerMock{typ: "default"}
			interceptor := GrpcPaymentValidationInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
			stream := &coverageServerStream{ctx: tc.ctx, recvErr: tc.recvErr, recvQueue: []*codec.GrpcFrame{{}}}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, func(any, grpc.ServerStream) error { t.Fatal("rejected request reached the provider"); return nil })
			require.Equal(t, tc.code, status.Code(err))
			if tc.recvErr != nil {
				require.ErrorIs(t, err, tc.recvErr)
			}
			require.False(t, paymentHandler.completeCalled)
			require.False(t, paymentHandler.completeAfterErrorCalled)
		})
	}
}

func TestStreamPaymentInterceptorUsesDefaultHandlerWithoutPaymentType(t *testing.T) {
	paymentHandler := &paymentHandlerMock{typ: "default"}
	interceptor := GrpcPaymentValidationInterceptor(&blockchain.ServiceMetadata{}, paymentHandler)
	stream := &coverageServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{}), recvQueue: []*codec.GrpcFrame{{}}}
	err := interceptor(nil, stream, &grpc.StreamServerInfo{}, func(any, grpc.ServerStream) error { return nil })
	require.NoError(t, err)
	require.True(t, paymentHandler.completeCalled)
}
