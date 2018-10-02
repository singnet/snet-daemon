package blockchain

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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
	var err error

	h.jobAddressBytes, err = getBytes(h.md, JobAddressHeader)
	if err != nil {
		return err
	}

	h.jobSignatureBytes, err = getBytes(h.md, JobSignatureHeader)
	if err != nil {
		return err
	}

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
