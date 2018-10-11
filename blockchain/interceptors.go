package blockchain

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
	// JobPaymentType each call should be payed using unique instance of funded Job
	JobPaymentType = "job"
	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	EscrowPaymentType = "escrow"
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
func GrpcStreamInterceptor(processor *Processor, jobHandler PaymentHandler, escrowHandler PaymentHandler) grpc.StreamServerInterceptor {
	if !processor.enabled {
		log.Info("Blockchain is disabled: no payment validation")
		return noOpInterceptor
	}

	log.Info("Blockchain is enabled: instantiate payment validation interceptor")
	interceptor := &paymentValidationInterceptor{
		jobPaymentHandler:    jobHandler,
		escrowPaymentHandler: escrowHandler,
	}
	return interceptor.intercept

}

type paymentValidationInterceptor struct {
	jobPaymentHandler    PaymentHandler
	escrowPaymentHandler PaymentHandler
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

	paymentType := JobPaymentType
	if ok && len(paymentTypeMd) > 0 {
		paymentType = paymentTypeMd[0]
	}
	log.WithField("paymentType", paymentType).Debug("Getting payment handler for the paymentType")

	switch {
	case paymentType == JobPaymentType:
		return interceptor.jobPaymentHandler, nil
	case paymentType == EscrowPaymentType:
		return interceptor.escrowPaymentHandler, nil
	default:
		log.WithField("paymentType", paymentType).Error("Unexpected payment type")
		return nil, status.Newf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}
}

func getBigInt(md metadata.MD, key string) (value *big.Int, err *status.Status) {
	str, err := getSingleValue(md, key)
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

func getBytes(md metadata.MD, key string) (result []byte, err *status.Status) {
	if !strings.HasSuffix(key, "-bin") {
		return nil, status.Newf(codes.InvalidArgument, "incorrect binary key name \"%v\"", key)
	}

	str, err := getSingleValue(md, key)
	if err != nil {
		return
	}

	return []byte(str), nil
}

func getBytesFromHexString(md metadata.MD, key string) (value []byte, err *status.Status) {
	str, err := getSingleValue(md, key)
	if err != nil {
		return
	}
	return common.FromHex(str), nil
}

func getSingleValue(md metadata.MD, key string) (value string, err *status.Status) {
	array := md.Get(key)

	if len(array) == 0 {
		return "", status.Newf(codes.InvalidArgument, "missing \"%v\"", key)
	}

	if len(array) > 1 {
		return "", status.Newf(codes.InvalidArgument, "too many values for key \"%v\": %v", key, array)
	}

	return array[0], nil
}

func noOpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}
