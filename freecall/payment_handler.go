package freecall

import (
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// PaymentType selects free-call authorization in the payment interceptor.
const PaymentType = "free-call"

type paymentHandler struct {
	service                  UserService
	freeCallPaymentValidator *PaymentValidator
	orgMetadata              *blockchain.OrganizationMetaData
	serviceMetadata          *blockchain.ServiceMetadata
}

// NewPaymentHandler verifies the caller request and token against the free-call
// signer from service metadata before checking the local quota. Complete records
// a successful call; CompleteAfterError releases the user lock without charging
// usage. Wallet balance is checked at token issuance, not on every service call,
// and no MPE claim is created.
func NewPaymentHandler(
	freeCallService UserService, processor blockchain.Processor, metadata *blockchain.OrganizationMetaData,
	pServiceMetaData *blockchain.ServiceMetadata) handler.StreamPaymentHandler {
	return &paymentHandler{
		service:         freeCallService,
		orgMetadata:     metadata,
		serviceMetadata: pServiceMetaData,
		freeCallPaymentValidator: NewPaymentValidator(processor.CurrentBlock,
			pServiceMetaData.FreeCallSignerAddress(), nil, config.GetTrustedFreeCallSignersAddresses()),
	}
}

func (h *paymentHandler) Type() (typ string) {
	return PaymentType
}

func (h *paymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	e := h.freeCallPaymentValidator.Validate(internalPayment)
	if e != nil {
		return nil, handler.PaymentErrorToGRPCError(e)
	}

	transaction, e := h.service.StartTransaction(internalPayment)
	if e != nil {
		return nil, handler.PaymentErrorToGRPCError(e)
	}

	return transaction, nil
}

func (h *paymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *Payment, err *handler.GrpcError) {

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

	parsedToken, blockExpiration, err2 := ParseToken(authToken)
	if err2 != nil {
		zap.L().Debug(err2.Error())
		return nil, handler.NewGrpcErrorf(codes.InvalidArgument, "invalid token: %v", err2)
	}

	return &Payment{
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

func (h *paymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return handler.PaymentErrorToGRPCError(payment.(Transaction).Commit())
}

func (h *paymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return handler.PaymentErrorToGRPCError(payment.(Transaction).Rollback())
}
