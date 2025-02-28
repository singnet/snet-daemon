package escrow

import (
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/handler"
)

const (

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	FreeCallPaymentType = "free-call"
)

type freeCallPaymentHandler struct {
	service                  FreeCallUserService
	freeCallPaymentValidator *FreeCallPaymentValidator
	orgMetadata              *blockchain.OrganizationMetaData
	serviceMetadata          *blockchain.ServiceMetadata
}

// NewPaymentHandler returns new MultiPartyEscrow contract payment handler.
func FreeCallPaymentHandler(
	freeCallService FreeCallUserService, processor blockchain.Processor, metadata *blockchain.OrganizationMetaData,
	pServiceMetaData *blockchain.ServiceMetadata) handler.StreamPaymentHandler {
	return &freeCallPaymentHandler{
		service:         freeCallService,
		orgMetadata:     metadata,
		serviceMetadata: pServiceMetaData,
		freeCallPaymentValidator: NewFreeCallPaymentValidator(processor.CurrentBlock,
			pServiceMetaData.FreeCallSignerAddress()),
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

	transaction, e := h.service.StartFreeCallUserTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (h *freeCallPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *FreeCallPayment, err *handler.GrpcError) {

	organizationId := config.GetString(config.OrganizationId)
	serviceId := config.GetString(config.ServiceId)

	userID, err := handler.GetSingleValue(context.MD, handler.FreeCallUserIdHeader)
	if err != nil {
		return
	}

	blockNumber, err := handler.GetBigInt(context.MD, handler.CurrentBlockNumberHeader)
	if err != nil {
		return
	}

	authTokenExpiryDate, err := handler.GetBigInt(context.MD, handler.FreeCallAuthTokenExpiryBlockNumberHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(context.MD, handler.PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	authToken, err := handler.GetBytes(context.MD, handler.FreeCallAuthTokenHeader)
	if err != nil {
		return
	}

	return &FreeCallPayment{
		OrganizationId:             organizationId,
		ServiceId:                  serviceId,
		UserId:                     userID,
		CurrentBlockNumber:         blockNumber,
		Signature:                  signature,
		AuthTokenExpiryBlockNumber: authTokenExpiryDate,
		AuthToken:                  authToken,
		GroupId:                    h.orgMetadata.GetGroupIdString(),
	}, nil
}

func (h *freeCallPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*freeCallTransaction).Commit())
}

func (h *freeCallPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*freeCallTransaction).Rollback())
}
