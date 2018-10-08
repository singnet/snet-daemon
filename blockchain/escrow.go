package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"time"
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
)

// TODO: add formatters for PaymentChannelKey, PaymentChannelData

// PaymentChannelKey specifies the channel in MultiPartyEscrow contract. It
// consists of two parts: channel id and channel nonce. Channel nonce is
// incremented each time when amount of tokens in channel descreases. Nonce
// allows reusing channel id without risk of overexpenditure.
type PaymentChannelKey struct {
	ID    *big.Int
	Nonce *big.Int
}

// PaymentChannelState is a current state of a payment channel. Payment
// channel may be in Open or Closed state.
type PaymentChannelState int

const (
	// Open means that channel is open and can be used to pay for calls.
	Open PaymentChannelState = 0
	// Closed means that channel is closed cannot be used to pay for calls.
	Closed PaymentChannelState = 1
)

// PaymentChannelData is to keep all channel related information.
type PaymentChannelData struct {
	// State is a payment channel state: Open or Closed.
	State PaymentChannelState
	// Sender is an Ethereum address of the client which created the channel.
	// It is and address to be charged for RPC call.
	Sender common.Address
	// FullAmount is an amount which is deposited in channel by Sender.
	FullAmount *big.Int
	// Expiration is an date and time at which channel will be expired. Since
	// this moment Sender can withdraw tokens from channel.
	Expiration time.Time
	// AuthorizedAmount is current amount which Sender authorized to withdraw by
	// service provider. This amount increments on price after each successful
	// RPC call.
	AuthorizedAmount *big.Int
	// Signature is a signature of last message containing Authorized amount.
	// It is required to claim tokens from channel.
	Signature []byte
}

// PaymentChannelStorage is an interface to get channel information by channel
// id.
type PaymentChannelStorage interface {
	// Get returns channel information by channel id.
	Get(key *PaymentChannelKey) (paymentChannel *PaymentChannelData, err error)
	// Put writes channel information by channel id.
	Put(key *PaymentChannelKey, state *PaymentChannelData) (err error)
	// CompareAndSwap atomically replaces old payment channel state by new
	// state.
	CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (err error)
}

// IncomeData is used to pass information to the pricing validation system.
// This system can use information about call to calculate price and verify
// income received.
type IncomeData struct {
	// Income is a difference between previous authorized amount and amount
	// which was received with current call.
	Income *big.Int
}

// IncomeValidator uses pricing information to check that call was payed
// correctly by channel sender. This interface can be implemented differently
// depending on pricing policy. For instance one can verify that call is payed
// according to invoice. Each RPC method can have different price and so on. To
// implement this strategies additional information from gRPC context can be
// required. In such case it should be added into IncomeData.
type IncomeValidator interface {
	// Validate returns nil if validation is successful or correct gRPC status
	// to be sent to client in case of validation error.
	Validate(*IncomeData) (err *status.Status)
}

// escrowPaymentHandler implements paymentHandlerType interface
type escrowPaymentHandler struct {
	storage         PaymentChannelStorage
	processor       *Processor
	incomeValidator IncomeValidator
	callContext     *callContextType
}

func newEscrowPaymentHandler(processor *Processor, storage PaymentChannelStorage, incomeValidator IncomeValidator, callContext *callContextType) *escrowPaymentHandler {
	return &escrowPaymentHandler{
		processor:       processor,
		storage:         storage,
		incomeValidator: incomeValidator,
		callContext:     callContext,
	}
}

type paymentType struct {
	channelKey *PaymentChannelKey
	amount     *big.Int
	signature  []byte
}

func (h *escrowPaymentHandler) validate() (err *status.Status) {
	payment, err := h.getPaymentFromMetadata()
	if err != nil {
		return err
	}
	return h.validatePaymentInternal(payment)
}

func (h *escrowPaymentHandler) validatePaymentInternal(payment *paymentType) (err *status.Status) {
	var log = log.WithField("payment", payment)

	paymentChannel, e := h.storage.Get(payment.channelKey)
	if e != nil {
		log.WithError(e).Warn("Payment channel not found")
		// TODO: job.go code always returns codes.Unauthenticated when
		// validations fails
		return status.Newf(codes.FailedPrecondition, "payment channel \"%v\" not found", payment.channelKey)
	}
	log = log.WithField("paymentChannel", paymentChannel)

	if paymentChannel.State != Open {
		log.Warn("Payment channel is not opened")
		return status.Newf(codes.FailedPrecondition, "payment channel \"%v\" is not opened", payment.channelKey)
	}

	signerAddress, err := h.getSignerAddressFromPayment(payment)
	if err != nil {
		return
	}

	if *signerAddress != paymentChannel.Sender {
		log.WithField("signerAddress", signerAddress).Warn("Channel sender is not equal to payment signer")
		return status.New(codes.Unauthenticated, "payment is not signed by channel sender")
	}

	now := time.Now()
	if paymentChannel.Expiration.Before(now) {
		log.WithField("now", now).Warn("Channel is expired")
		return status.Newf(codes.FailedPrecondition, "payment channel is expired since \"%v\"", paymentChannel.Expiration)
	}

	if paymentChannel.FullAmount.Cmp(payment.amount) < 0 {
		log.Warn("Not enough tokens on payment channel")
		return status.Newf(codes.FailedPrecondition, "not enough tokens on payment channel, channel amount: %v, payment amount: %v ", paymentChannel.FullAmount, payment.amount)
	}

	income := big.NewInt(0)
	income.Sub(payment.amount, paymentChannel.AuthorizedAmount)
	err = h.incomeValidator.Validate(&IncomeData{Income: income})
	if err != nil {
		return
	}

	// TODO: current job code comletes payment iff service returned no error
	e = h.storage.CompareAndSwap(
		payment.channelKey,
		paymentChannel,
		&PaymentChannelData{
			State:            paymentChannel.State,
			FullAmount:       paymentChannel.FullAmount,
			Expiration:       paymentChannel.Expiration,
			AuthorizedAmount: payment.amount,
			Signature:        payment.signature,
		},
	)
	if e != nil {
		log.WithError(e).Error("Unable to store new payment channel state")
		return status.New(codes.Internal, "unable to store new payment channel state")
	}

	return nil
}

func (h *escrowPaymentHandler) getSignerAddressFromPayment(payment *paymentType) (signer *common.Address, err *status.Status) {
	paymentHash := crypto.Keccak256(
		hashPrefix32Bytes,
		crypto.Keccak256(
			h.processor.escrowContractAddress.Bytes(),
			bigIntToBytes(payment.channelKey.ID),
			bigIntToBytes(payment.channelKey.Nonce),
			bigIntToBytes(payment.amount),
		),
	)

	publicKey, e := crypto.SigToPub(paymentHash, payment.signature)
	if e != nil {
		log.WithError(e).WithFields(log.Fields{
			"paymentHash": common.ToHex(paymentHash),
			"publicKey":   publicKey,
			"err":         err,
		}).Warn("Incorrect signature")
		return nil, status.New(codes.Unauthenticated, "payment signature is not valid")
	}

	keyOwnerAddress := crypto.PubkeyToAddress(*publicKey)
	return &keyOwnerAddress, nil
}

func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func (h *escrowPaymentHandler) getPaymentFromMetadata() (payment *paymentType, err *status.Status) {
	channelID, err := getBigInt(h.callContext.md, PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := getBigInt(h.callContext.md, PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	amount, err := getBigInt(h.callContext.md, PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := getBytes(h.callContext.md, PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &paymentType{&PaymentChannelKey{channelID, channelNonce}, amount, signature}, nil
}

func (h *escrowPaymentHandler) complete(err error) error {
	return err
}
