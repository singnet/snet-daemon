package escrow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
)

const (
	// PaymentChannelIDHeader is a MultiPartyEscrow contract payment channel
	// id. Value is a string containing a decimal number.
	PaymentChannelIDHeader = "snet-payment-channel-id"
	// PaymentChannelNonceHeader is a payment channel nonce value. Value is a
	// string containing a decimal number.
	PaymentChannelNonceHeader = "snet-payment-channel-nonce"
	// PaymentChannelAmountHeader is an amount of payment channel value
	// which server is authorized to withdraw after handling the RPC call.
	// Value is a string containing a decimal number.
	PaymentChannelAmountHeader = "snet-payment-channel-amount"
	// PaymentChannelSignatureHeader is a signature of the client to confirm
	// amount withdrawing authorization. Value is an array of bytes.
	PaymentChannelSignatureHeader = "snet-payment-channel-signature-bin"

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	EscrowPaymentType = "escrow"
)

type paymentChannelPaymentHandler struct {
	service            PaymentChannelService
	mpeContractAddress func() common.Address
	incomeValidator    IncomeValidator
}

// NewPaymentHandler retuns new MultiPartyEscrow contract payment handler.
func NewPaymentHandler(
	service PaymentChannelService,
	processor *blockchain.Processor,
	incomeValidator IncomeValidator) handler.PaymentHandler {
	return &paymentChannelPaymentHandler{
		service:            service,
		mpeContractAddress: processor.EscrowContractAddress,
		incomeValidator:    incomeValidator,
	}
}

func (h *paymentChannelPaymentHandler) Type() (typ string) {
	return EscrowPaymentType
}

func (h *paymentChannelPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *status.Status) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	transaction, e := h.service.StartPaymentTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcStatus(e)
	}

	income := big.NewInt(0)
	income.Sub(internalPayment.Amount, transaction.Channel().AuthorizedAmount)
	err = h.incomeValidator.Validate(&IncomeData{Income: income, GrpcContext: context})
	if err != nil {
		return
	}

	return transaction, nil
}

func (h *paymentChannelPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *Payment, err *status.Status) {
	channelID, err := handler.GetBigInt(context.MD, PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(context.MD, PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	amount, err := handler.GetBigInt(context.MD, PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(context.MD, PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &Payment{
		MpeContractAddress: h.mpeContractAddress(),
		ChannelID:          channelID,
		ChannelNonce:       channelNonce,
		Amount:             amount,
		Signature:          signature,
	}, nil
}

func (h *paymentChannelPaymentHandler) Validate(_payment handler.Payment) (err *status.Status) {
	return nil
}

func (h *paymentChannelPaymentHandler) Complete(payment handler.Payment) (err *status.Status) {
	return paymentErrorToGrpcStatus(payment.(*paymentTransaction).Commit())
}

func (h *paymentChannelPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *status.Status) {
	return paymentErrorToGrpcStatus(payment.(*paymentTransaction).Rollback())
}

func paymentErrorToGrpcStatus(err error) *status.Status {
	if err == nil {
		return nil
	}

	if _, ok := err.(*PaymentError); !ok {
		return status.Newf(codes.Internal, "internal error: %v", err)
	}

	var grpcCode codes.Code
	switch err.(*PaymentError).Code {
	case Internal:
		grpcCode = codes.Internal
	case Unauthenticated:
		grpcCode = codes.Unauthenticated
	case FailedPrecondition:
		grpcCode = codes.FailedPrecondition
	default:
		grpcCode = codes.Internal
	}

	return status.Newf(grpcCode, err.(*PaymentError).Message)
}
