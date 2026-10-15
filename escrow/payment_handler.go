package escrow

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/lifecycle"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/singnet/snet-daemon/v6/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

const (

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	EscrowPaymentType = "escrow"
)

type paymentChannelPaymentHandler struct {
	background         *lifecycle.Group
	service            PaymentChannelService
	mpeContractAddress func() common.Address
	incomeValidator    IncomeStreamValidator
	currentBlock       func() (*big.Int, error)
}

// NewPaymentHandler returns new MultiPartyEscrow contract payment handler.
func NewPaymentHandler(
	service PaymentChannelService,
	processor blockchain.Processor,
	incomeValidator IncomeStreamValidator, background ...*lifecycle.Group) handler.StreamPaymentHandler {
	h := &paymentChannelPaymentHandler{
		service:            service,
		mpeContractAddress: processor.EscrowContractAddress,
		currentBlock:       processor.CurrentBlock,
		incomeValidator:    incomeValidator,
	}
	if len(background) > 0 {
		h.background = background[0]
	}
	return h
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
	e = h.incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: context})
	if e != nil {
		// Make sure the transaction is rolled back, else this will cause a lock on the channel
		transaction.Rollback()
		return nil, paymentErrorToGrpcError(e)
	}

	return transaction, nil
}

func (h *paymentChannelPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *Payment, err *handler.GrpcError) {
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

	signature, err := handler.GetBytes(context.MD, handler.PaymentChannelSignatureHeader)
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
	if err = paymentErrorToGrpcError(payment.(*paymentTransaction).Commit()); err == nil {
		publishChannelStats(h.background, payment, h.currentBlock)
	}
	return err
}

func publishChannelStats(background *lifecycle.Group, payment handler.Payment, currentBlock func() (*big.Int, error)) {
	if config.GetBool(config.MeteringEnabled) {
		background.Go(func(ctx context.Context) { PublishChannelStatsContext(ctx, payment, currentBlock) })
	}
}

func PublishChannelStats(payment handler.Payment, currentBlock func() (*big.Int, error)) (grpcErr *handler.GrpcError) {
	return PublishChannelStatsContext(context.Background(), payment, currentBlock)
}

func PublishChannelStatsContext(ctx context.Context, payment handler.Payment, currentBlock func() (*big.Int, error)) (grpcErr *handler.GrpcError) {
	if !config.GetBool(config.MeteringEnabled) {
		return nil
	}

	paymentTransaction, ok := payment.(*paymentTransaction)
	if !ok {
		return nil
	}

	channelStats := &metrics.ChannelStats{ChannelId: paymentTransaction.payment.ChannelID,
		AuthorizedAmount: paymentTransaction.payment.Amount,
		FullAmount:       paymentTransaction.Channel().FullAmount,
		Nonce:            paymentTransaction.Channel().Nonce,
		GroupID:          utils.BytesToBase64(paymentTransaction.Channel().GroupID[:]),
	}
	meteringURL := config.GetString(config.MeteringEndpoint) + "/contract-api/channel/" + channelStats.ChannelId.String() + "/balance"

	channelStats.OrganizationID = config.GetString(config.OrganizationId)
	channelStats.ServiceID = config.GetString(config.ServiceId)
	zap.L().Debug("Payment channel payment handler is publishing channel statistics", zap.Any("ChannelStats", channelStats))
	commonStats := &metrics.CommonStats{
		GroupID: channelStats.GroupID, UserName: paymentTransaction.Channel().Sender.Hex()}

	block, err := currentBlock()
	if err != nil {
		return handler.NewGrpcErrorf(codes.Internal, "Unable to get latest block")
	}
	status := metrics.PublishContext(ctx, channelStats, meteringURL, commonStats, block)
	if !status {
		zap.L().Warn("Payment handler unable to post latest off-chain Channel state on contract API Endpoint for metering", zap.String("meteringURL", meteringURL))
		return handler.NewGrpcErrorf(codes.Internal, "Unable to publish status error")
	}
	return nil
}

func (h *paymentChannelPaymentHandler) CompleteAfterError(payment handler.Payment, result error) (err *handler.GrpcError) {
	return paymentErrorToGrpcError(payment.(*paymentTransaction).Rollback())
}

func paymentErrorToGrpcError(err error) *handler.GrpcError {
	return handler.PaymentErrorToGRPCError(err)
}
