package escrow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/handler"
)

const (
	TrainPaymentType = "train-call"
)

type trainUnaryPaymentHandler struct {
	service            PaymentChannelService
	mpeContractAddress func() common.Address
	incomeValidator    IncomeUnaryValidator
	currentBlock       func() (*big.Int, error)
}

type trainStreamPaymentHandler struct {
	service            PaymentChannelService
	mpeContractAddress func() common.Address
	currentBlock       func() (*big.Int, error)
	incomeValidator    IncomeStreamValidator
}

func (t trainStreamPaymentHandler) Type() (typ string) {
	return TrainPaymentType
}

func (t trainStreamPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := t.getPaymentFromContext(context.MD)
	if err != nil {
		return
	}

	transaction, e := t.service.StartPaymentTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	income := big.NewInt(0)
	income.Sub(internalPayment.Amount, transaction.Channel().AuthorizedAmount)
	e = t.incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: context})
	if e != nil {
		//Make sure the transaction is Rolled back , else this will cause a lock on the channel
		transaction.Rollback()
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (t trainStreamPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	if err = paymentErrorToGrpcError(payment.(*paymentTransaction).Commit()); err == nil {
		go PublishChannelStats(payment, t.currentBlock)
	}
	return err
}

func (t trainStreamPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*paymentTransaction).Rollback())
}

func (t trainStreamPaymentHandler) getPaymentFromContext(md metadata.MD) (payment *Payment, err *handler.GrpcError) {
	channelID, err := handler.GetBigInt(md, handler.PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(md, handler.PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	amount, err := handler.GetBigInt(md, handler.PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(md, handler.PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &Payment{
		MpeContractAddress: t.mpeContractAddress(),
		ChannelID:          channelID,
		ChannelNonce:       channelNonce,
		Amount:             amount,
		Signature:          signature,
	}, nil
}

// NewTrainUnaryPaymentHandler returns new MultiPartyEscrow contract payment handler.
func NewTrainUnaryPaymentHandler(
	service PaymentChannelService,
	processor blockchain.Processor,
	incomeValidator IncomeUnaryValidator) handler.UnaryPaymentHandler {
	return &trainUnaryPaymentHandler{
		service:            service,
		mpeContractAddress: processor.EscrowContractAddress,
		currentBlock:       processor.CurrentBlock,
		incomeValidator:    incomeValidator,
	}
}

// NewTrainStreamPaymentHandler returns new MultiPartyEscrow contract payment handler.
func NewTrainStreamPaymentHandler(
	service PaymentChannelService,
	processor blockchain.Processor,
	incomeValidator IncomeStreamValidator) handler.StreamPaymentHandler {
	return &trainStreamPaymentHandler{
		service:            service,
		mpeContractAddress: processor.EscrowContractAddress,
		currentBlock:       processor.CurrentBlock,
		incomeValidator:    incomeValidator,
	}
}

func (h *trainUnaryPaymentHandler) Type() (typ string) {
	return TrainPaymentType
}

func (h *trainUnaryPaymentHandler) Payment(context *handler.GrpcUnaryContext) (payment handler.Payment, err *handler.GrpcError) {
	internalPayment, err := h.getPaymentFromContext(context.MD)
	if err != nil {
		return
	}

	transaction, e := h.service.StartPaymentTransaction(internalPayment)
	if e != nil {
		return nil, paymentErrorToGrpcError(e)
	}

	income := big.NewInt(0)
	zap.L().Debug("[trainUnaryPaymentHandler.Payment]", zap.Any("Amount", internalPayment.Amount), zap.Any("AuthorizedAmount", transaction.Channel().AuthorizedAmount))
	income.Sub(internalPayment.Amount, transaction.Channel().AuthorizedAmount)
	e = h.incomeValidator.Validate(&IncomeUnaryData{Income: income, GrpcContext: context})
	if e != nil {
		//Make sure the transaction is Rolled back , else this will cause a lock on the channel
		transaction.Rollback()
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (h *trainUnaryPaymentHandler) getPaymentFromContext(md metadata.MD) (payment *Payment, err *handler.GrpcError) {
	channelID, err := handler.GetBigInt(md, handler.PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(md, handler.PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	amount, err := handler.GetBigInt(md, handler.PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(md, handler.PaymentChannelSignatureHeader)
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

func (h *trainUnaryPaymentHandler) Complete(payment handler.Payment) (err *handler.GrpcError) {
	if err = paymentErrorToGrpcError(payment.(*paymentTransaction).Commit()); err == nil {
		go PublishChannelStats(payment, h.currentBlock)
	}
	return err
}

func (h *trainUnaryPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*paymentTransaction).Rollback())
}
