package pricing

import (
	"context"
	"errors"
	"math"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/codec"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestDynamicMethodPrice_GetPriceType(t *testing.T) {
	priceType := DynamicMethodPrice{}

	assert.Equal(t, DYNAMIC_PRICING, priceType.GetPriceType())
	assert.Equal(t, "dynamic_pricing", priceType.GetPriceType())
}

type pricingTransportStream struct{ method string }

func (s pricingTransportStream) Method() string             { return s.method }
func (pricingTransportStream) SetHeader(metadata.MD) error  { return nil }
func (pricingTransportStream) SendHeader(metadata.MD) error { return nil }
func (pricingTransportStream) SetTrailer(metadata.MD) error { return nil }

type pricingInputStream struct {
	grpc.ServerStream
	ctx  context.Context
	data []byte
}

func (s *pricingInputStream) Context() context.Context { return s.ctx }
func (s *pricingInputStream) RecvMsg(message any) error {
	message.(*codec.GrpcFrame).Data = s.data
	return nil
}

func pricingContext(t *testing.T, ctx context.Context) *handler.GrpcStreamContext {
	t.Helper()
	stream, err := handler.NewWrapperServerStream(&pricingInputStream{ctx: ctx, data: []byte("request")}, ctx)
	require.NoError(t, err)
	return &handler.GrpcStreamContext{InStream: stream}
}

func pricingConfig(t *testing.T, endpoint string) {
	t.Helper()
	previous := config.Vip()
	v := viper.New()
	v.Set(config.ServiceEndpointKey, endpoint)
	v.Set(config.MaxMessageSizeInMB, 4)
	v.Set(config.EnableDynamicPricing, true)
	config.SetVip(v)
	t.Cleanup(func() { config.SetVip(previous) })
}

func TestDynamicPricingForwardsRequestAndMetadata(t *testing.T) {
	for _, amount := range []uint64{0, 42, math.MaxUint64} {
		t.Run(new(big.Int).SetUint64(amount).String(), func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			type receivedRequest struct {
				method string
				data   []byte
				md     metadata.MD
			}
			received := make(chan receivedRequest, 1)
			server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
				frame := &codec.GrpcFrame{}
				if err := stream.RecvMsg(frame); err != nil {
					return err
				}
				method, _ := grpc.MethodFromServerStream(stream)
				md, _ := metadata.FromIncomingContext(stream.Context())
				received <- receivedRequest{method, frame.Data, md}
				return stream.SendMsg(&PriceInCogs{Price: amount})
			}))
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			t.Cleanup(func() {
				server.Stop()
				err := <-done
				if !errors.Is(err, grpc.ErrServerStopped) {
					require.NoError(t, err)
				}
			})
			pricingConfig(t, "http://"+listener.Addr().String())
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "test-token"))
			ctx = grpc.NewContextWithServerTransportStream(ctx, pricingTransportStream{"/example.Service/Predict"})
			derived := pricingContext(t, ctx)
			priceType := DynamicMethodPrice{serviceMetaData: &blockchain.ServiceMetadata{
				DynamicPriceMethodMapping: map[string]string{"/example.Service/Predict": "/example.Service/Price"},
			}}
			price, err := priceType.GetPrice(derived)
			require.NoError(t, err)
			require.Equal(t, new(big.Int).SetUint64(amount), price)
			request := <-received
			require.Equal(t, "/example.Service/Price", request.method)
			require.Equal(t, []byte("request"), request.data)
			require.Equal(t, []string{"test-token"}, request.md.Get("authorization"))
			frame := &codec.GrpcFrame{}
			require.NoError(t, derived.InStream.RecvMsg(frame))
			require.Equal(t, request.data, frame.Data, "price estimation must preserve the request for inference")
		})
	}
}

func TestDynamicPricingRejectsInvalidContext(t *testing.T) {
	for _, scenario := range []string{"missing method", "invalid endpoint", "missing metadata", "missing mapping", "canceled context"} {
		t.Run(scenario, func(t *testing.T) {
			pricingConfig(t, "http://127.0.0.1:1")
			ctx := context.Context(t.Context())
			if scenario != "missing method" {
				ctx = grpc.NewContextWithServerTransportStream(ctx, pricingTransportStream{"/example.Service/Predict"})
			}
			if scenario != "missing metadata" {
				ctx = metadata.NewIncomingContext(ctx, metadata.MD{})
			}
			if scenario == "invalid endpoint" {
				config.Vip().Set(config.ServiceEndpointKey, "://invalid")
			}
			mapping := map[string]string{"/example.Service/Predict": "/example.Service/Price"}
			if scenario == "missing mapping" {
				mapping = nil
			}
			if scenario == "canceled context" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			priceType := DynamicMethodPrice{serviceMetaData: &blockchain.ServiceMetadata{DynamicPriceMethodMapping: mapping}}
			price, err := priceType.GetPrice(pricingContext(t, ctx))
			require.Error(t, err)
			require.Nil(t, price)
			if scenario == "canceled context" {
				require.Equal(t, codes.Canceled, status.Code(err))
			}
		})
	}
}

type pricingClientStream struct {
	grpc.ClientStream
	sendErr, recvErr error
	response         []byte
	sent             any
	received         bool
}

func (s *pricingClientStream) SendMsg(message any) error { s.sent = message; return s.sendErr }
func (s *pricingClientStream) RecvMsg(message any) error {
	s.received = true
	message.(*codec.GrpcFrame).Data = s.response
	return s.recvErr
}

func TestDynamicPricingResponseErrors(t *testing.T) {
	failure := errors.New("transport failed")
	for _, scenario := range []string{"send", "receive", "invalid protobuf"} {
		t.Run(scenario, func(t *testing.T) {
			stream := &pricingClientStream{response: []byte{0xff}}
			if scenario == "send" {
				stream.sendErr = failure
			}
			if scenario == "receive" {
				stream.recvErr = failure
			}
			priceType := DynamicMethodPrice{}
			derived := pricingContext(t, t.Context())
			price, err := priceType.getPriceFromPricingMethod(derived.InStream, stream)
			require.Nil(t, price)
			if scenario == "invalid protobuf" {
				require.Error(t, err)
			} else {
				require.ErrorIs(t, err, failure)
			}
			require.Equal(t, []byte("request"), stream.sent.(*codec.GrpcFrame).Data)
			require.Equal(t, scenario != "send", stream.received)
		})
	}
}
