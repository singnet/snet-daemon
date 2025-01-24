package escrow

import (
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/handler"
	"github.com/singnet/snet-daemon/v5/pricing"
	"github.com/singnet/snet-daemon/v5/token"
	"go.uber.org/zap"
)

const (
	PrePaidPaymentType = "prepaid-call"
)

type PrePaidPaymentValidator struct {
	priceStrategy *pricing.PricingStrategy
	tokenManager  token.Manager
}

func NewPrePaidPaymentValidator(pricing *pricing.PricingStrategy, manager token.Manager) *PrePaidPaymentValidator {
	return &PrePaidPaymentValidator{
		priceStrategy: pricing,
		tokenManager:  manager,
	}
}

type PrePaidPaymentHandler struct {
	service                 PrePaidService
	PrePaidPaymentValidator *PrePaidPaymentValidator
	orgMetadata             *blockchain.OrganizationMetaData
	serviceMetadata         *blockchain.ServiceMetadata
}

func (validator *PrePaidPaymentValidator) Validate(payment *PrePaidPayment) (err error) {
	//Validate the token
	return validator.tokenManager.VerifyToken(payment.AuthToken, payment.ChannelID)
}

// NewPaymentHandler returns new MultiPartyEscrow contract payment handler.
func NewPrePaidPaymentHandler(
	PrePaidService PrePaidService, metadata *blockchain.OrganizationMetaData,
	pServiceMetaData *blockchain.ServiceMetadata, pricing *pricing.PricingStrategy, manager token.Manager) handler.StreamPaymentHandler {
	return &PrePaidPaymentHandler{
		service:                 PrePaidService,
		orgMetadata:             metadata,
		serviceMetadata:         pServiceMetaData,
		PrePaidPaymentValidator: NewPrePaidPaymentValidator(pricing, manager),
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
	// Increment the used amount
	if err := h.service.UpdateUsage(prePaidPayment.ChannelID, price, USED_AMOUNT); err != nil {
		return nil, paymentErrorToGrpcError(err)
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

	authToken, err := handler.GetSingleValue(context.MD, handler.PrePaidAuthTokenHeader)
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

// Just logging , as we increase the usage before calling the service
// assuming the service call will be successful
func (h *PrePaidPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	prePaidTransaction := payment.(PrePaidTransaction)
	zap.L().Debug("usage successfully updated and state of channel is consistent",
		zap.Any("price", prePaidTransaction.Price()),
		zap.Any("channelID", prePaidTransaction.ChannelId()))
	return nil
}

func (h *PrePaidPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	//we need to Keep track of the amount charged for service that errored
	//and refund back  !!
	prePaidTransaction := payment.(PrePaidTransaction)
	if err = paymentErrorToGrpcError(h.service.UpdateUsage(prePaidTransaction.ChannelId(),
		prePaidTransaction.Price(), REFUND_AMOUNT)); err != nil {
		zap.L().Error(err.Err().Error())
		zap.L().Error("usage INCONSISTENT state on Channel, usage wrongly increased", zap.Any("usage", prePaidTransaction.Price()),
			zap.Any("ChannelID", prePaidTransaction.ChannelId()))
	}

	zap.L().Debug("usage on channel id was updated already, however the refund state has been adjusted accordingly",
		zap.Any("usage", prePaidTransaction.Price()), zap.Any("channelID", prePaidTransaction.ChannelId()))
	return err
}
