package escrow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"

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

func (h *paymentChannelPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	transaction, e := h.service.StartPaymentTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	income := big.NewInt(0)
	income.Sub(internalPayment.Amount, transaction.Channel().AuthorizedAmount)
	e = h.incomeValidator.Validate(&IncomeData{Income: income, GrpcContext: context})
	if e != nil {
		//Make sure the transaction is Rolled back , else this will cause a lock on the channel
		transaction.Rollback()
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (h *paymentChannelPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *Payment, err *handler.GrpcError) {
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

func (h *paymentChannelPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*paymentTransaction).Commit())
}

func (h *paymentChannelPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*paymentTransaction).Rollback())
}

func paymentErrorToGrpcError(err error) *handler.GrpcError {
	if err == nil {
		return nil
	}

	if _, ok := err.(*PaymentError); !ok {
		return handler.NewGrpcErrorf(codes.Internal, "internal error: %v", err)
	}

	var grpcCode codes.Code
	switch err.(*PaymentError).Code {
	case Internal:
		grpcCode = codes.Internal
	case Unauthenticated:
		grpcCode = codes.Unauthenticated
	case FailedPrecondition:
		grpcCode = codes.FailedPrecondition
	case IncorrectNonce:
		grpcCode = handler.IncorrectNonce
	default:
		grpcCode = codes.Internal
	}

	return handler.NewGrpcErrorf(grpcCode, err.(*PaymentError).Message)
}
