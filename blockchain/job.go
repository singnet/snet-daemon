package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
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
