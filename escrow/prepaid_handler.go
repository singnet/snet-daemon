package escrow

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"github.com/singnet/snet-daemon/pricing"
	log "github.com/sirupsen/logrus"
)

const (
	PrePaidPaymentType = "prepaid-call"
)

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
	if err != nil {
		return nil, err
	}
	price, priceError := h.PrePaidPaymentValidator.priceStrategy.GetPrice(context)
	if priceError != nil {
		return nil, paymentErrorToGrpcError(priceError)
	}
	validateErr := h.PrePaidPaymentValidator.Validate(prePaidPayment)
	if validateErr != nil {
		return nil, paymentErrorToGrpcError(validateErr)
	}
	//Increment the used amount
	if err := h.service.UpdateUsage(prePaidPayment.ChannelID, price, USED_AMOUNT); err != nil {
		return nil, paymentErrorToGrpcError(validateErr)
	}
	transaction = &prePaidTransactionImpl{price: price, channelId: prePaidPayment.ChannelID}
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

//Just logging , as we increase the usage before calling the service
//assuming the service call will be successful
func (h *PrePaidPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	prePaidTransaction := payment.(PrePaidTransaction)
	log.Debugf("usage %v successfully updated and channel id: %v state is consistent",
		prePaidTransaction.Price(),
		prePaidTransaction.ChannelId())
	return nil
}

func (h *PrePaidPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	//we need to Keep track of the amount charged for service that errored
	//and refund back  !!
	prePaidTransaction := payment.(PrePaidTransaction)
	if err = paymentErrorToGrpcError(h.service.UpdateUsage(prePaidTransaction.ChannelId(),
		prePaidTransaction.Price(), REFUND_AMOUNT)); err != nil {
		log.Error(err)
		log.Errorf("usage INCONSISTENT state on Channel id:%v , usage wrongly increased by %v",
			prePaidTransaction.ChannelId(), prePaidTransaction.Price())
	}
	log.Debugf("usage:%v on channel id:%v was updated already, however the refund state has been"+
		"adjusted accordingly", prePaidTransaction.Price(), prePaidTransaction.ChannelId())
	return err
}
