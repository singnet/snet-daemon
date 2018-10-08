package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
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

type GrpcStreamContext struct {
	MD   metadata.MD
	Info *grpc.StreamServerInfo
}

func (p Processor) paymentValidationInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	var err *status.Status

	context, err := getGrpcContext(ss, info)
	if err != nil {
		return err.Err()
	}

	paymentHandler, err := p.getPaymentHandler(context)
	if err != nil {
		return err.Err()
	}

	err = paymentHandler.validate()
	if err != nil {
		return err.Err()
	}

	e := handler(srv, ss)
	if e != nil {
		paymentHandler.completeAfterError(e)
		return e
	}

	paymentHandler.complete()
	return nil
}

func getGrpcContext(serverStream grpc.ServerStream, info *grpc.StreamServerInfo) (context *GrpcStreamContext, err *status.Status) {
	md, ok := metadata.FromIncomingContext(serverStream.Context())
	if !ok {
		return nil, status.New(codes.InvalidArgument, "missing metadata")
	}

	return &GrpcStreamContext{
		MD:   md,
		Info: info,
	}, nil
}

func (p Processor) getPaymentHandler(callContext *GrpcStreamContext) (handler paymentHandlerType, err *status.Status) {
	paymentTypeMd, ok := callContext.MD[PaymentTypeHeader]

	paymentType := JobPaymentType
	if ok && len(paymentTypeMd) > 0 {
		paymentType = paymentTypeMd[0]
	}

	switch {
	case paymentType == JobPaymentType:
		return newJobPaymentHandler(&p, callContext), nil
	case paymentType == EscrowPaymentType:
		return newEscrowPaymentHandler(&p, nil, nil, callContext), nil
	default:
		return nil, status.Newf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}
}

type paymentHandlerType interface {
	validate() *status.Status
	complete()
	completeAfterError(error)
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
