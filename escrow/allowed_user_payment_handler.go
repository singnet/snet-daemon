package escrow

import (
	"github.com/singnet/snet-daemon/handler"
)

const (

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	AllowedUserPaymentType = "allowed-user"
)

type AllowedUserPayment struct {
	// Signature passed
	Signature []byte
}

type allowedUserPaymentHandler struct {
	validator *AllowedUserPaymentValidator
}

func AllowedUserPaymentHandler() handler.PaymentHandler {
	return &allowedUserPaymentHandler{
		validator: &AllowedUserPaymentValidator{},
	}
}

func (h *allowedUserPaymentHandler) Type() (typ string) {
	return AllowedUserPaymentType
}

func (h *allowedUserPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	e := h.validator.Validate(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	return internalPayment, nil
}

func (h *allowedUserPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *AllowedUserPayment, err *handler.GrpcError) {

	signature, err := handler.GetBytes(context.MD, handler.AllowedUserSignatureHeader)
	if err != nil {
		return
	}
	return &AllowedUserPayment{
		Signature: signature,
	}, nil
}

func (h *allowedUserPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return nil
}

func (h *allowedUserPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return nil
}
