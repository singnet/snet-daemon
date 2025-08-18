package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/ctxkeys"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type GrpcUnaryContext struct {
	MD   metadata.MD
	Info *grpc.UnaryServerInfo
}

// SenderProvider allows retrieving the sender's Ethereum address,
// independent of the specific type from pkg/escrow.
type SenderProvider interface {
	GetSender() common.Address
}

type UnaryPaymentHandler interface {
	// Type is a content of the PaymentTypeHeader field that triggers usage of the
	// payment handler.
	Type() (typ string)
	// Payment extracts payment data from gRPC request context and checks
	// validity of payment data. It returns nil if data is valid or
	// appropriate gRPC status otherwise.
	Payment(context *GrpcUnaryContext) (payment Payment, err *GrpcError)
	// Complete completes payment if the service successfully processed the gRPC call
	Complete(payment Payment) (err *GrpcError)
	// CompleteAfterError completes payment if service returns error.
	CompleteAfterError(payment Payment, result error) (err *GrpcError)
}

type paymentValidationUnaryInterceptor struct {
	serviceMetadata       *blockchain.ServiceMetadata
	defaultPaymentHandler UnaryPaymentHandler
	paymentHandlers       map[string]UnaryPaymentHandler
}

func (interceptor *paymentValidationUnaryInterceptor) unaryIntercept(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, e error) {
	var err *GrpcError

	ctx = context.WithValue(ctx, ctxkeys.MethodKey, info.FullMethod)

	lastSlash := strings.LastIndex(info.FullMethod, "/")
	methodName := info.FullMethod[lastSlash+1:]

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		zap.L().Error("Invalid metadata", zap.Any("info", info))
		return nil, NewGrpcError(codes.InvalidArgument, "missing metadata").Err()
	}

	// pass non-training requests and free requests
	if methodName != "validate_model" && methodName != "train_model" {
		ctx = metadata.NewIncomingContext(ctx, md)
		resp, e := handler(ctx, req)
		if e != nil {
			zap.L().Warn("gRPC handler returned error", zap.Error(e))
			return resp, e
		}
		return resp, e
	}

	c := &GrpcUnaryContext{MD: md.Copy(), Info: info}

	zap.L().Debug("[unaryIntercept] grpc metadata", zap.Any("md", c.MD))
	zap.L().Debug("[unaryIntercept] New gRPC call received", zap.Any("context", c))

	paymentHandler, err := interceptor.getPaymentHandler(c)
	if err != nil {
		return nil, err.Err()
	}

	payment, err := paymentHandler.Payment(c)
	if err != nil {
		return nil, err.Err()
	}

	if sp, ok := payment.(SenderProvider); ok {
		outMD := c.MD.Copy()
		ethAddr := sp.GetSender().Hex()
		outMD.Set(SnetUserAddressHeader, ethAddr)
		outMD.Set("snet-daemon-debug", "unaryIntercept")
		ctx = metadata.NewIncomingContext(ctx, outMD)
	}

	defer func() {
		if r := recover(); r != nil {
			zap.L().Warn("Service handler called panic(panicValue)", zap.Any("panicValue", r))
			paymentHandler.CompleteAfterError(payment, fmt.Errorf("service handler called panic(%v)", r))
			panic("re-panic after payment handler error handling")
		} else if e == nil {
			err = paymentHandler.Complete(payment)
			if err != nil {
				// return err.Err()
				e = err.Err()
			}
		} else {
			err = paymentHandler.CompleteAfterError(payment, e)
			if err != nil {
				// return err.Err()
				e = err.Err()
			}
		}
	}()

	zap.L().Debug("[unaryIntercept] New payment received", zap.Any("payment", payment))

	resp, e = handler(ctx, req)
	if e != nil {
		zap.L().Warn("gRPC handler returned error", zap.Error(e))
		return resp, e
	}

	return resp, e
}

func (interceptor *paymentValidationUnaryInterceptor) getPaymentHandler(context *GrpcUnaryContext) (handler UnaryPaymentHandler, err *GrpcError) {
	paymentTypeMd, ok := context.MD[PaymentTypeHeader]
	if !ok || len(paymentTypeMd) == 0 {
		zap.L().Debug("Payment type was not set by caller, return default payment handler",
			zap.String("defaultPaymentHandlerType", interceptor.defaultPaymentHandler.Type()))
		return interceptor.defaultPaymentHandler, nil
	}

	paymentType := paymentTypeMd[0]
	zap.L().Debug("Payment metadata", zap.String("paymentType", paymentType), zap.Any("paymentTypeMd", paymentTypeMd))
	paymentHandler, ok := interceptor.paymentHandlers[paymentType]
	if !ok {
		zap.L().Error("Unexpected payment type", zap.String("paymentType", paymentType))
		return nil, NewGrpcErrorf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}

	zap.L().Debug("Return payment handler by type", zap.Any("paymentType", paymentType))
	return paymentHandler, nil
}

func GrpcPaymentValidationUnaryInterceptor(serviceData *blockchain.ServiceMetadata, defaultPaymentHandler UnaryPaymentHandler, paymentHandler ...UnaryPaymentHandler) grpc.UnaryServerInterceptor {
	interceptor := &paymentValidationUnaryInterceptor{
		defaultPaymentHandler: defaultPaymentHandler,
		paymentHandlers:       make(map[string]UnaryPaymentHandler),
		serviceMetadata:       serviceData,
	}

	interceptor.paymentHandlers[defaultPaymentHandler.Type()] = defaultPaymentHandler
	zap.L().Info("Default payment handler registered", zap.Any("defaultPaymentType", defaultPaymentHandler.Type()))
	for _, handler := range paymentHandler {
		interceptor.paymentHandlers[handler.Type()] = handler
		zap.L().Info("Payment handler for type registered", zap.Any("paymentType", handler.Type()))
	}
	return interceptor.unaryIntercept
}

// NoOpUnaryInterceptor is a gRPC interceptor that doesn't do payment checking.
func NoOpUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(ctx, req)
}
