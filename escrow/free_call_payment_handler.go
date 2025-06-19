package escrow

import (
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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

func FreeCallPaymentHandler(
	freeCallService FreeCallUserService, processor blockchain.Processor, metadata *blockchain.OrganizationMetaData,
	pServiceMetaData *blockchain.ServiceMetadata) handler.StreamPaymentHandler {
	return &freeCallPaymentHandler{
		service:         freeCallService,
		orgMetadata:     metadata,
		serviceMetadata: pServiceMetaData,
		freeCallPaymentValidator: NewFreeCallPaymentValidator(processor.CurrentBlock,
			pServiceMetaData.FreeCallSignerAddress(), nil, config.GetTrustedFreeCallSignersAddresses()),
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

	userID, _ := handler.GetSingleValue(context.MD, handler.FreeCallUserIdHeader)

	userAddress, err := handler.GetSingleValue(context.MD, handler.FreeCallUserAddressHeader)
	if err != nil {
		return
	}

	blockNumber, err := handler.GetBigInt(context.MD, handler.CurrentBlockNumberHeader)
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

	parsedToken, blockExpiration, err2 := ParseFreeCallToken(authToken)
	if err2 != nil {
		zap.L().Debug(err2.Error())
		return nil, handler.NewGrpcErrorf(codes.InvalidArgument, "invalid token: %v", err2)
	}

	return &FreeCallPayment{
		OrganizationId:             config.GetString(config.OrganizationId),
		ServiceId:                  config.GetString(config.ServiceId),
		UserID:                     userID,
		Address:                    userAddress,
		CurrentBlockNumber:         blockNumber,
		Signature:                  signature,
		AuthTokenExpiryBlockNumber: blockExpiration,
		AuthToken:                  authToken,
		AuthTokenParsed:            parsedToken,
		GroupId:                    h.orgMetadata.GetGroupIdString(),
	}, nil
}

func (h *freeCallPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*freeCallTransaction).Commit())
}

func (h *freeCallPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*freeCallTransaction).Rollback())
}
