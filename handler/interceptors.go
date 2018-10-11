package handler

import (
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

const (
	// PaymentTypeHeader is a type of payment used to pay for a RPC call.
	// Supported types are: "job", "escrow".
	PaymentTypeHeader = "snet-payment-type"
)

// GrpcStreamContext contains information about gRPC call which is used to
// validate payment and pricing.
type GrpcStreamContext struct {
	MD   metadata.MD
	Info *grpc.StreamServerInfo
}

// Payment represents payment handler specific data which is validated
// and used to complete payment.
type Payment interface{}

// PaymentHandler interface which is used by gRPC interceptor to get, validate
// and complete payment. There are two payment handler implementations so far:
// jobPaymentHandler and escrowPaymentHandler.
type PaymentHandler interface {
	// Type is a content of PaymentTypeHeader field which triggers usage of the
	// payment handler.
	Type() (typ string)
	// Payment extracts payment data from gRPC request context.
	Payment(context *GrpcStreamContext) (payment Payment, err *status.Status)
	// Validate checks validity of payment data, it returns nil if data is
	// valid or appropriate gRPC error otherwise.
	Validate(payment Payment) (err *status.Status)
	// Complete completes payment if gRPC call was successfully proceeded by
	// service.
	Complete(payment Payment) (err *status.Status)
	// CompleteAfterError completes payment if service returns error.
	CompleteAfterError(payment Payment, result error) (err *status.Status)
}

// GrpcStreamInterceptor returns gRPC interceptor to validate payment. If
// blockchain is disabled then noOpInterceptor is returned.
func GrpcStreamInterceptor(defaultPaymentHandler PaymentHandler, paymentHandler ...PaymentHandler) grpc.StreamServerInterceptor {
	interceptor := &paymentValidationInterceptor{
		defaultPaymentHandler: defaultPaymentHandler,
		paymentHandlers:       make(map[string]PaymentHandler),
	}

	interceptor.paymentHandlers[defaultPaymentHandler.Type()] = defaultPaymentHandler
	log.WithField("defaultPaymentType", defaultPaymentHandler.Type()).Info("Default payment handler registered")
	for _, handler := range paymentHandler {
		interceptor.paymentHandlers[handler.Type()] = handler
		log.WithField("paymentType", handler.Type()).Info("Payment handler for type registered")
	}

	return interceptor.intercept

}

type paymentValidationInterceptor struct {
	defaultPaymentHandler PaymentHandler
	paymentHandlers       map[string]PaymentHandler
}

func (interceptor *paymentValidationInterceptor) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	var err *status.Status

	context, err := getGrpcContext(ss, info)
	if err != nil {
		return err.Err()
	}
	log.WithField("context", context).Debug("New gRPC call received")

	paymentHandler, err := interceptor.getPaymentHandler(context)
	if err != nil {
		return err.Err()
	}

	payment, err := paymentHandler.Payment(context)
	if err != nil {
		return err.Err()
	}
	log.WithField("payment", payment).Debug("New payment received")

	err = paymentHandler.Validate(payment)
	if err != nil {
		return err.Err()
	}
	log.Debug("Payment validated")

	e := handler(srv, ss)
	if e != nil {
		log.WithError(e).Warn("gRPC handler returned error")
		err = paymentHandler.CompleteAfterError(payment, e)
		if err != nil {
			return err.Err()
		}
		return e
	}

	err = paymentHandler.Complete(payment)
	if err != nil {
		return err.Err()
	}
	log.Debug("Payment completed")

	return nil
}

func getGrpcContext(serverStream grpc.ServerStream, info *grpc.StreamServerInfo) (context *GrpcStreamContext, err *status.Status) {
	md, ok := metadata.FromIncomingContext(serverStream.Context())
	if !ok {
		log.WithField("info", info).Error("Invalid metadata")
		return nil, status.New(codes.InvalidArgument, "missing metadata")
	}

	return &GrpcStreamContext{
		MD:   md,
		Info: info,
	}, nil
}

func (interceptor *paymentValidationInterceptor) getPaymentHandler(context *GrpcStreamContext) (handler PaymentHandler, err *status.Status) {
	paymentTypeMd, ok := context.MD[PaymentTypeHeader]
	if !ok || len(paymentTypeMd) == 0 {
		log.WithField("defaultPaymentHandlerType", interceptor.defaultPaymentHandler.Type()).Debug("Payment type was not set by caller, return default payment handler")
		return interceptor.defaultPaymentHandler, nil
	}

	paymentType := paymentTypeMd[0]
	paymentHandler, ok := interceptor.paymentHandlers[paymentType]
	if !ok {
		log.WithField("paymentType", paymentType).Error("Unexpected payment type")
		return nil, status.Newf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}

	log.WithField("paymentType", paymentType).Debug("Return payment handler by type")
	return paymentHandler, nil
}

// GetBigInt gets big.Int value from gRPC metadata
func GetBigInt(md metadata.MD, key string) (value *big.Int, err *status.Status) {
	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}

	value = big.NewInt(0)
	e := value.UnmarshalText([]byte(str))
	if e != nil {
		return nil, status.Newf(codes.InvalidArgument, "incorrect format \"%v\": \"%v\"", key, str)
	}

	return
}

// GetBytes gets bytes array value from gRPC metadata for key with '-bin'
// suffix, internally this data is encoded as base64
func GetBytes(md metadata.MD, key string) (result []byte, err *status.Status) {
	if !strings.HasSuffix(key, "-bin") {
		return nil, status.Newf(codes.InvalidArgument, "incorrect binary key name \"%v\"", key)
	}

	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}

	return []byte(str), nil
}

// GetBytesFromHex gets bytes array value from gRPC metadata, bytes array is
// encoded as hex string
func GetBytesFromHex(md metadata.MD, key string) (value []byte, err *status.Status) {
	str, err := GetSingleValue(md, key)
	if err != nil {
		return
	}
	return common.FromHex(str), nil
}

// GetSingleValue gets string value from gRPC metadata
func GetSingleValue(md metadata.MD, key string) (value string, err *status.Status) {
	array := md.Get(key)

	if len(array) == 0 {
		return "", status.Newf(codes.InvalidArgument, "missing \"%v\"", key)
	}

	if len(array) > 1 {
		return "", status.Newf(codes.InvalidArgument, "too many values for key \"%v\": %v", key, array)
	}

	return array[0], nil
}

// NoOpInterceptor is a gRPC interceptor which doesn't do payment checking.
func NoOpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}
