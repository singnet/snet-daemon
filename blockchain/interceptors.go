package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
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
		return newJobPaymentHandler(p, md), nil
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

type jobPaymentHandler struct {
	p                 Processor
	md                metadata.MD
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

func newJobPaymentHandler(p Processor, md metadata.MD) *jobPaymentHandler {
	return &jobPaymentHandler{p: p, md: md}
}

func (h *jobPaymentHandler) validatePayment() error {
	jobAddressMd, ok := h.md[JobAddressHeader]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing snet-job-address")
	}

	h.jobAddressBytes = common.FromHex(jobAddressMd[0])

	jobSignatureMd, ok := h.md[JobSignatureHeader]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing snet-job-signature")
	}

	h.jobSignatureBytes = common.FromHex(jobSignatureMd[0])
	valid := h.p.IsValidJobInvocation(h.jobAddressBytes, h.jobSignatureBytes)
	if !valid {
		return status.Errorf(codes.Unauthenticated, "job invocation not valid")
	}

	return nil
}

func (h *jobPaymentHandler) completePayment(err error) error {
	if err == nil {
		h.p.CompleteJob(h.jobAddressBytes, h.jobSignatureBytes)
	}
	return err
}

func noOpInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	return handler(srv, ss)
}

func getBigInt(md metadata.MD, key string) (*big.Int, error) {
	array := md.Get(key)
	if len(array) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing \"%v\"", key)
	}

	value := big.NewInt(0)
	err := value.UnmarshalText([]byte(array[0]))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "incorrect format \"%v\": \"%v\"", key, array[0])
	}

	return value, nil
}

func getBytes(md metadata.MD, key string) ([]byte, error) {
	value := md.Get(key)
	if len(value) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing \"%v\"", key)
	}
	return common.FromHex(value[0]), nil
}
