package escrow

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"github.com/singnet/snet-daemon/pricing"
)

const (
	PrePaidPaymentType = "prepaid-call"
)

//todo
type PrePaidPaymentValidator struct {
	priceStrategy *pricing.PricingStrategy
}

func NewPrePaidPaymentValidator(pricing *pricing.PricingStrategy) *PrePaidPaymentValidator {
	return &PrePaidPaymentValidator{
		priceStrategy: pricing,
	}
}

type PrePaidPaymentHandler struct {
	service                 PrePaidService
	PrePaidPaymentValidator *PrePaidPaymentValidator
	orgMetadata             *blockchain.OrganizationMetaData
	serviceMetadata         *blockchain.ServiceMetadata
}

//todo
func (validator *PrePaidPaymentValidator) Validate(payment *PrePaidPayment) (err error) {
	//Validate the token

	//Ensure that the amount in Channel >= Signed Amount !!
	return nil
}

// NewPaymentHandler returns new MultiPartyEscrow contract payment handler.
func NewPrePaidPaymentHandler(
	PrePaidService PrePaidService, metadata *blockchain.OrganizationMetaData,
	pServiceMetaData *blockchain.ServiceMetadata, pricing *pricing.PricingStrategy) handler.PaymentHandler {
	return &PrePaidPaymentHandler{
		service:                 PrePaidService,
		orgMetadata:             metadata,
		serviceMetadata:         pServiceMetaData,
		PrePaidPaymentValidator: NewPrePaidPaymentValidator(pricing),
	}
}

func (h *PrePaidPaymentHandler) Type() (typ string) {
	return PrePaidPaymentType
}

func (h *PrePaidPaymentHandler) Payment(context *handler.GrpcStreamContext) (transaction handler.Payment,
	err *handler.GrpcError) {

	prePaidPayment, err := h.getPaymentFromContext(context)
	price, priceError := h.PrePaidPaymentValidator.priceStrategy.GetPrice(context)
	if priceError != nil {
		return nil, paymentErrorToGrpcError(priceError)
	}
	key := prePaidPayment.GetKey()

	//Check if Token is Valid
	validateErr := h.PrePaidPaymentValidator.Validate(prePaidPayment)
	if validateErr != nil {
		return nil, paymentErrorToGrpcError(validateErr)
	}
	//Increment the used amount
	if err := h.service.UpdateUsage(key, IncreaseUsedAmount, price); err != nil {
		return nil, paymentErrorToGrpcError(validateErr)
	}
	transaction = &prePaidTransactionImpl{price: price, key: key}
	return transaction, nil
}

func (h *PrePaidPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *PrePaidPayment,
	err *handler.GrpcError) {

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

func (h *PrePaidPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return nil
}

//todo
func (h *PrePaidPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	//we need to decrement the used amount ( used amount = used amount - price ) as the service errored  !!
	prePaidTransaction := payment.(PrePaidTransaction)
	return paymentErrorToGrpcError(h.service.UpdateUsage(prePaidTransaction.PrePaidKey(), DecreaseUsedAmount, prePaidTransaction.Price()))
}
