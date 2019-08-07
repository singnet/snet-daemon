package escrow

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
)

const (

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	FreeCallPaymentType = "free-call"
)

type freeCallPaymentHandler struct {
	freeCallPaymentValidator *FreeCallPaymentValidator
}


// NewPaymentHandler retuns new MultiPartyEscrow contract payment handler.
func FreeCallPaymentHandler(
	processor *blockchain.Processor) handler.PaymentHandler {
	return &freeCallPaymentHandler{
		freeCallPaymentValidator: NewFreeCallPaymentValidator(processor.CurrentBlock),
	}
}

func (h *freeCallPaymentHandler) Type() (typ string) {
	return FreeCallPaymentType
}

func (h *freeCallPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	e := h.freeCallPaymentValidator.Validate(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	return internalPayment, nil
}

func (h *freeCallPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *FreeCallPayment, err *handler.GrpcError) {

	organizationId , err := handler.GetSingleValue(context.MD, config.GetString(config.OrganizationId))
	if err != nil {
		return
	}

	serviceId , err := handler.GetSingleValue(context.MD, config.GetString(config.ServiceId))
	if err != nil {
		return
	}

	userID , err := handler.GetSingleValue(context.MD, handler.FreeCallUserIdHeader)
	if err != nil {
		return
	}

	blockNumber,err := handler.GetBigInt(context.MD,handler.CurrentBlockNumberHeader)
	if err != nil {
		return
	}

	signature,err := handler.GetBytes(context.MD, handler.PaymentChannelSignatureHeader)
	if err != nil {
		return
	}


	return &FreeCallPayment{
		OrganizationId:organizationId,
		ServiceId:serviceId,
		UserId:userID,
		CurrentBlockNumber:blockNumber,
		Signature:          signature,
	}, nil
}

func (h *freeCallPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return nil //todo , will make a call to metering service indicating success state of the service call
}

func (h *freeCallPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return nil // todo , will make a call to the metering service indicating Failed state of the service call
}

