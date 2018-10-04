package blockchain

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"math/big"
	"time"
)

const (
	// PaymentChannelIdHeader is a MultiPartyEscrow contract payment channel id
	PaymentChannelIdHeader = "snet-payment-channel-id"
	// PaymentChannelNonceHeader is a payment channel nonce value
	PaymentChannelNonceHeader = "snet-payment-channel-nonce"
	// PaymentChannelAmountHeader is an amount of payment channel value
	// which server is authorized withdraw after handling the RPC call.
	PaymentChannelAmountHeader = "snet-payment-channel-amount"
	// PaymentChannelSignatureHeader is a signature of the client to confirm
	// authorized amount
	PaymentChannelSignatureHeader = "snet-payment-channel-signature"
)

// TODO: add formatters for PaymentChannelKey, PaymentChannelData

type PaymentChannelKey struct {
	Id    *big.Int
	Nonce *big.Int
}

type PaymentChannelState int

const (
	Open   PaymentChannelState = 0
	Closed PaymentChannelState = 1
)

type PaymentChannelData struct {
	State              PaymentChannelState
	FullAmount         *big.Int
	ExpirationDateTime time.Time
	AuthorizedAmount   *big.Int
	ClientSignature    []byte
}

type PaymentChannelStorage interface {
	Get(key *PaymentChannelKey) (*PaymentChannelData, error)
	CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) error
}

// escrowPaymentHandler implements paymentHandlerType interface
type escrowPaymentHandler struct {
	md        metadata.MD
	storage   PaymentChannelStorage
	processor *Processor
}

func newEscrowPaymentHandler() *escrowPaymentHandler {
	return &escrowPaymentHandler{}
}

type paymentData struct {
	channelKey *PaymentChannelKey
	amount     *big.Int
	signature  []byte
}

func (h *escrowPaymentHandler) validatePayment() error {
	payment, err := h.getPaymentFromMetadata()
	if err != nil {
		return err
	}
	return h.validatePaymentInternal(payment)
}

func (h *escrowPaymentHandler) getPaymentFromMetadata() (*paymentData, error) {
	/*
		id, err := getBigInt(h.md, PaymentChannelIdHeader)
		if err != nil {
			return err
		}

		nonce, err := getBigInt(h.md, PaymentChannelNonceHeader)
		if err != nil {
			return err
		}

		PaymentChannelData := h.storage.Get(&PaymentChannelKey{id, nonce})

		signature, err := getBytes(h.md, PaymentChannelSignatureHeader)
	*/
	return nil, status.Errorf(codes.Unimplemented, "not implemented yet")
}

func (h *escrowPaymentHandler) validatePaymentInternal(payment *paymentData) error {
	var err error
	var log = log.WithField("payment", payment)

	err = h.checkPaymentSignature(payment)
	if err != nil {
		return err
	}

	paymentChannel, err := h.storage.Get(payment.channelKey)
	if err != nil {
		log.WithError(err).Warn("Payment channel not found")
		return status.Errorf(codes.FailedPrecondition, "payment channel \"%v\" not found", payment.channelKey)
	}
	log = log.WithField("paymentChannel", paymentChannel)

	if paymentChannel.State != Open {
		log.Warn("Payment channel is not opened")
		return status.Errorf(codes.FailedPrecondition, "payment channel \"%v\" is not opened", payment.channelKey)
	}

	now := time.Now()
	if paymentChannel.ExpirationDateTime.Before(now) {
		log.WithField("now", now).Warn("Channel is expired")
		return status.Errorf(codes.FailedPrecondition, "payment channel is expired since \"%v\"", paymentChannel.ExpirationDateTime)
	}

	if paymentChannel.FullAmount.Cmp(payment.amount) < 0 {
		log.Warn("Not enough tokens on payment channel")
		return status.Errorf(codes.FailedPrecondition, "not enough tokens on payment channel, channel amount: %v, payment amount: %v ", paymentChannel.FullAmount, payment.amount)
	}

	price, err := h.processor.agent.CurrentPrice(
		&bind.CallOpts{
			Pending: true,
			From:    common.HexToAddress(h.processor.address),
		})
	if err != nil {
		log.WithError(err).Error("Cannot get current price from Agent")
		return status.Errorf(codes.Internal, "cannot get current price from Agent")
	}

	nextAuthorizedAmount := big.NewInt(0)
	nextAuthorizedAmount.Add(paymentChannel.AuthorizedAmount, price)
	if nextAuthorizedAmount.Cmp(payment.amount) != 0 {
		log.Warn("Next authorized amount is not equal to previous amount plus price")
		return status.Errorf(codes.FailedPrecondition, "Next authorized amount is not equal to previous amount plus price, previous amount: \"%v\", price: \"%v\", new amount: \"%v\"", paymentChannel.AuthorizedAmount, price, payment.amount)
	}

	err = h.storage.CompareAndSwap(
		payment.channelKey,
		paymentChannel,
		&PaymentChannelData{
			State:              paymentChannel.State,
			FullAmount:         paymentChannel.FullAmount,
			ExpirationDateTime: paymentChannel.ExpirationDateTime,
			AuthorizedAmount:   payment.amount,
			ClientSignature:    payment.signature,
		},
	)
	if err != nil {
		log.WithError(err).Error("Unable to store new payment channel state")
		return status.Error(codes.Internal, "unable to store new payment channel state")
	}

	return nil
}

func (h *escrowPaymentHandler) checkPaymentSignature(payment *paymentData) error {
	return status.Errorf(codes.Unimplemented, "not implemented yet")
}

func (h *escrowPaymentHandler) completePayment(err error) error {
	return err
}
