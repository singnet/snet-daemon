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

func (p Processor) jobValidationInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {

	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	paymentHandler, err := p.newPaymentHandlerByType(md)
	if err != nil {
		return err
	}

	err = paymentHandler.validatePayment()
	if err != nil {
		return err
	}

	err = handler(srv, ss)

	return paymentHandler.completePayment(err)
}

func (p Processor) newPaymentHandlerByType(md metadata.MD) (paymentHandlerType, error) {
	paymentTypeMd, ok := md[PaymentTypeHeader]

	paymentType := JobPaymentType
	if ok && len(paymentTypeMd) > 0 {
		paymentType = paymentTypeMd[0]
	}

	switch {
	case paymentType == JobPaymentType:
		return newJobPaymentHandler(&p, md), nil
	case paymentType == EscrowPaymentType:
		return newEscrowPaymentHandler(), nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentType)
	}
}

type paymentHandlerType interface {
	validatePayment() error
	completePayment(error) error
}

func getBigInt(md metadata.MD, key string) (value *big.Int, err error) {
	str, err := getSingleValue(md, key)
	if err != nil {
		return
	}

	value = big.NewInt(0)
	err = value.UnmarshalText([]byte(str))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "incorrect format \"%v\": \"%v\"", key, str)
	}

	return
}

func getBytes(md metadata.MD, key string) (result []byte, err error) {
	if !strings.HasSuffix(key, "-bin") {
		return nil, status.Errorf(codes.InvalidArgument, "incorrect binary key name \"%v\"", key)
	}

	str, err := getSingleValue(md, key)
	if err != nil {
		return
	}

	return []byte(str), nil
}

func getBytesFromHexString(md metadata.MD, key string) (value []byte, err error) {
	str, err := getSingleValue(md, key)
	if err != nil {
		return
	}
	return common.FromHex(str), nil
}

func getSingleValue(md metadata.MD, key string) (value string, err error) {
	array := md.Get(key)

	if len(array) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "missing \"%v\"", key)
	}

	if len(array) > 1 {
		return "", status.Errorf(codes.InvalidArgument, "too many values for key \"%v\": %v", key, array)
	}

	return array[0], nil
}

func noOpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}
