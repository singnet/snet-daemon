package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (p Processor) jobValidationInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {

	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.InvalidArgument, "missing metadata")
	}

	paymentTypeMd, ok := md[PaymentTypeHeader]
	paymentType := ""
	if ok && len(paymentTypeMd) > 0 {
		paymentType = paymentTypeMd[0]
	}

	var paymentHandler paymentHandlerType
	switch {
	case !ok || paymentType == JobPaymentType:
		paymentHandler = newJobPaymentHandler(p, md)
	default:
		return status.Errorf(codes.InvalidArgument, "unexpected \"%v\", value: \"%v\"", PaymentTypeHeader, paymentTypeMd)
	}

	valid, err := paymentHandler.validatePayment()
	if !valid {
		return err
	}

	err = handler(srv, ss)

	return paymentHandler.completePayment(err)
}

type paymentHandlerType interface {
	validatePayment() (bool, error)
	completePayment(error) error
}

type jobPaymentHandler struct {
	p                 Processor
	md                metadata.MD
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

func newJobPaymentHandler(p Processor, md metadata.MD) paymentHandlerType {
	return &jobPaymentHandler{p: p, md: md}
}

func (h *jobPaymentHandler) validatePayment() (bool, error) {
	jobAddressMd, ok := h.md[JobAddressHeader]
	if !ok {
		return false, status.Errorf(codes.InvalidArgument, "missing snet-job-address")
	}

	h.jobAddressBytes = common.FromHex(jobAddressMd[0])

	jobSignatureMd, ok := h.md[JobSignatureHeader]
	if !ok {
		return false, status.Errorf(codes.InvalidArgument, "missing snet-job-signature")
	}

	h.jobSignatureBytes = common.FromHex(jobSignatureMd[0])
	valid := h.p.IsValidJobInvocation(h.jobAddressBytes, h.jobSignatureBytes)
	if !valid {
		return false, status.Errorf(codes.Unauthenticated, "job invocation not valid")
	}

	return true, nil
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
