package escrow

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
)

const (
	PrePaidPaymentType = "prepaid-call"
)

//todo
type PrePaidPaymentValidator struct {
}

func NewPrePaidPaymentValidator() *PrePaidPaymentValidator {
	return &PrePaidPaymentValidator{}
}

type PrePaidPaymentHandler struct {
	service                 PrePaidUserService
	PrePaidPaymentValidator *PrePaidPaymentValidator
	orgMetadata             *blockchain.OrganizationMetaData
	serviceMetadata         *blockchain.ServiceMetadata
}

//todo
func (validator *PrePaidPaymentValidator) Validate(payment *PrePaidPayment) (err error) {
	//Validate the token
	//lock the state
	//Check if used amount + Price <= planned amount
	//update used amount
	//Release the lock on state
	return nil
}

// NewPaymentHandler returns new MultiPartyEscrow contract payment handler.
func NewPrePaidPaymentHandler(
	PrePaidService PrePaidUserService, metadata *blockchain.OrganizationMetaData,
	pServiceMetaData *blockchain.ServiceMetadata) handler.PaymentHandler {
	return &PrePaidPaymentHandler{
		service:                 PrePaidService,
		orgMetadata:             metadata,
		serviceMetadata:         pServiceMetaData,
		PrePaidPaymentValidator: NewPrePaidPaymentValidator(),
	}
}

func (h *PrePaidPaymentHandler) Type() (typ string) {
	return PrePaidPaymentType
}

func (h *PrePaidPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	e := h.PrePaidPaymentValidator.Validate(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	transaction, e := h.service.StartPrePaidUserTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (h *PrePaidPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *PrePaidPayment, err *handler.GrpcError) {

	organizationId := config.GetString(config.OrganizationId)
	channelID, err := handler.GetBigInt(context.MD, handler.PaymentChannelIDHeader)
	if err != nil {
		return
	}

	authToken, err := handler.GetBytes(context.MD, handler.PrePaidAuthTokenHeader)
	if err != nil {
		return
	}

	return &PrePaidPayment{
		ChannelID:      channelID,
		OrganizationId: organizationId,
		GroupId:        h.orgMetadata.GetGroupIdString(),
		AuthToken:      authToken,
	}, nil
}

//todo
func (h *PrePaidPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return nil
}

//todo
func (h *PrePaidPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	//we need to decrement the used amount ( used amount = used amount - price ) as the service errored  !!
	return nil
}
