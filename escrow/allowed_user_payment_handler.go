package escrow

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/utils"
	"google.golang.org/grpc/codes"
)

type allowedUserPaymentHandler struct {
	validator *AllowedUserPaymentValidator
}

func AllowedUserPaymentHandler() handler.StreamPaymentHandler {
	return &allowedUserPaymentHandler{
		validator: &AllowedUserPaymentValidator{},
	}
}

// clients should be oblivious to this handler
func (h *allowedUserPaymentHandler) Type() (typ string) {
	return EscrowPaymentType
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

func (h *allowedUserPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *Payment, err *handler.GrpcError) {
	channelID, err := handler.GetBigInt(context.MD, handler.PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(context.MD, handler.PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	amount, err := handler.GetBigInt(context.MD, handler.PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	address, err := handler.GetSingleValue(context.MD, handler.PaymentMultiPartyEscrowAddressHeader)
	if err != nil {
		return
	}
	if !common.IsHexAddress(address) {
		err = handler.NewGrpcErrorf(codes.InvalidArgument, "Address is not a valid Hex address \"%v\": %v",
			handler.PaymentMultiPartyEscrowAddressHeader, address)
		return
	}

	signature, err := handler.GetBytes(context.MD, handler.PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &Payment{
		ChannelID:          channelID,
		ChannelNonce:       channelNonce,
		Amount:             amount,
		Signature:          signature,
		MpeContractAddress: utils.ToChecksumAddress(address),
	}, nil
}

func (h *allowedUserPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return nil
}

func (h *allowedUserPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return nil
}
