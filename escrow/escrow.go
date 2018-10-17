package escrow

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
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

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	EscrowPaymentType = "escrow"
)

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
	// Nonce is a nonce of this channel state
	Nonce *big.Int
	// State is a payment channel state: Open or Closed.
	State PaymentChannelState
	// Sender is an Ethereum address of the client which created the channel.
	// It is and address to be charged for RPC call.
	Sender common.Address
	// Recipient is an address which can claim funds from channel using
	// signature. It is an address of service provider.
	Recipient common.Address
	// GroupId is an id of the group of service replicas which share the same
	// payment channel.
	GroupId *big.Int
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
	// Get returns channel information by channel id. ok value indicates
	// whether passed key was found. err indicates storage error.
	Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error)
	// Put writes channel information by channel id.
	Put(key *PaymentChannelKey, state *PaymentChannelData) (err error)
	// CompareAndSwap atomically replaces old payment channel state by new
	// state. If ok flag is true and err is nil then operation was successful.
	// If err is nil and ok is false then operation failed because prevState is
	// not equal to current state. err indicates storage error.
	CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error)
}

func (key PaymentChannelKey) String() string {
	return fmt.Sprintf("{ID: %v, Nonce: %v}", key.ID, key.Nonce)
}

func (state PaymentChannelState) String() string {
	return [...]string{
		"Open",
		"Closed",
	}[state]
}

func (key PaymentChannelData) String() string {
	return fmt.Sprintf("{State: %v, Sender: %v, FullAmount: %v, Expiration: %v, AuthorizedAmount: %v, Signature: %v",
		key.State, blockchain.AddressToHex(&key.Sender), key.FullAmount, key.Expiration.Format(time.RFC3339), key.AuthorizedAmount, blockchain.BytesToBase64(key.Signature))
}

// IncomeData is used to pass information to the pricing validation system.
// This system can use information about call to calculate price and verify
// income received.
type IncomeData struct {
	// Income is a difference between previous authorized amount and amount
	// which was received with current call.
	Income *big.Int
	// GrpcContext contains gRPC stream context information. For instance
	// metadata could be used to pass invoice id to check pricing.
	GrpcContext *handler.GrpcStreamContext
}

// IncomeValidator uses pricing information to check that call was payed
// correctly by channel sender. This interface can be implemented differently
// depending on pricing policy. For instance one can verify that call is payed
// according to invoice. Each RPC method can have different price and so on. To
// implement this strategies additional information from gRPC context can be
// required. In such case it should be added into handler.GrpcStreamContext.
type IncomeValidator interface {
	// Validate returns nil if validation is successful or correct gRPC status
	// to be sent to client in case of validation error.
	Validate(*IncomeData) (err *status.Status)
}

// escrowPaymentHandler implements paymentHandlerType interface
type escrowPaymentHandler struct {
	escrowContractAddress common.Address
	storage               PaymentChannelStorage
	incomeValidator       IncomeValidator
}

// NewEscrowPaymentHandler returns instance of handler.PaymentHandler to validate
// payments via MultiPartyEscrow contract.
func NewEscrowPaymentHandler(processor *blockchain.Processor, storage PaymentChannelStorage, incomeValidator IncomeValidator) handler.PaymentHandler {
	return &escrowPaymentHandler{
		escrowContractAddress: processor.EscrowContractAddress(),
		storage:               storage,
		incomeValidator:       incomeValidator,
	}
}

type escrowPaymentType struct {
	grpcContext *handler.GrpcStreamContext
	channelKey  *PaymentChannelKey
	amount      *big.Int
	signature   []byte
	channel     *PaymentChannelData
}

func (p *escrowPaymentType) String() string {
	return fmt.Sprintf("{grpcContext: %v, channelKey: %v, amount: %v, signature: %v, channel: %v}",
		p.grpcContext, p.channelKey, p.amount, blockchain.BytesToBase64(p.signature), p.channel)
}

func (h *escrowPaymentHandler) Type() (typ string) {
	return EscrowPaymentType
}

func (h *escrowPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *status.Status) {
	channelID, err := handler.GetBigInt(context.MD, PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(context.MD, PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	channelKey := &PaymentChannelKey{channelID, channelNonce}
	channel, ok, e := h.storage.Get(channelKey)
	if e != nil {
		return nil, status.Newf(codes.Internal, "payment channel storage error")
	}
	if !ok {
		log.Warn("Payment channel not found")
		return nil, status.Newf(codes.InvalidArgument, "payment channel \"%v\" not found", channelKey)
	}

	amount, err := handler.GetBigInt(context.MD, PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(context.MD, PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &escrowPaymentType{
		grpcContext: context,
		channelKey:  channelKey,
		amount:      amount,
		signature:   signature,
		channel:     channel,
	}, nil
}

func (h *escrowPaymentHandler) Validate(_payment handler.Payment) (err *status.Status) {
	var payment = _payment.(*escrowPaymentType)
	var log = log.WithField("payment", payment)

	if payment.channel.State != Open {
		log.Warn("Payment channel is not opened")
		return status.Newf(codes.Unauthenticated, "payment channel \"%v\" is not opened", payment.channelKey)
	}

	signerAddress, err := h.getSignerAddressFromPayment(payment)
	if err != nil {
		return
	}

	if *signerAddress != payment.channel.Sender {
		log.WithField("signerAddress", blockchain.AddressToHex(signerAddress)).Warn("Channel sender is not equal to payment signer")
		return status.New(codes.Unauthenticated, "payment is not signed by channel sender")
	}

	now := time.Now()
	if payment.channel.Expiration.Before(now) {
		log.WithField("now", now).Warn("Channel is expired")
		return status.Newf(codes.Unauthenticated, "payment channel is expired since \"%v\"", payment.channel.Expiration)
	}

	if payment.channel.FullAmount.Cmp(payment.amount) < 0 {
		log.Warn("Not enough tokens on payment channel")
		return status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: %v, payment amount: %v", payment.channel.FullAmount, payment.amount)
	}

	income := big.NewInt(0)
	income.Sub(payment.amount, payment.channel.AuthorizedAmount)
	err = h.incomeValidator.Validate(&IncomeData{Income: income, GrpcContext: payment.grpcContext})
	if err != nil {
		return
	}

	return
}

func (h *escrowPaymentHandler) getSignerAddressFromPayment(payment *escrowPaymentType) (signer *common.Address, err *status.Status) {
	paymentHash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(
			h.escrowContractAddress.Bytes(),
			bigIntToBytes(payment.channelKey.ID),
			bigIntToBytes(payment.channelKey.Nonce),
			bigIntToBytes(payment.amount),
		),
	)

	log := log.WithFields(log.Fields{
		"payment":     payment,
		"paymentHash": common.ToHex(paymentHash),
	})

	v, _, _, e := blockchain.ParseSignature(payment.signature)
	if e != nil {
		log.WithError(e).Warn("Error parsing signature")
		return nil, status.New(codes.Unauthenticated, "payment signature is not valid")
	}

	signature := bytes.Join([][]byte{payment.signature[0:64], {v % 27}}, nil)
	publicKey, e := crypto.SigToPub(paymentHash, signature)
	if e != nil {
		log.WithError(e).WithField("signature", signature).Warn("Incorrect signature")
		return nil, status.New(codes.Unauthenticated, "payment signature is not valid")
	}

	keyOwnerAddress := crypto.PubkeyToAddress(*publicKey)
	return &keyOwnerAddress, nil
}

func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func (h *escrowPaymentHandler) Complete(_payment handler.Payment) (err *status.Status) {
	var payment = _payment.(*escrowPaymentType)
	ok, e := h.storage.CompareAndSwap(
		payment.channelKey,
		payment.channel,
		&PaymentChannelData{
			State:            payment.channel.State,
			Sender:           payment.channel.Sender,
			FullAmount:       payment.channel.FullAmount,
			Expiration:       payment.channel.Expiration,
			AuthorizedAmount: payment.amount,
			Signature:        payment.signature,
		},
	)
	if e != nil {
		log.WithError(e).Error("Unable to store new payment channel state")
		return status.New(codes.Internal, "unable to store new payment channel state")
	}
	if !ok {
		log.WithField("payment", payment).Warn("Channel state was changed concurrently")
		return status.Newf(codes.Unauthenticated, "state of payment channel \"%v\" was concurrently updated", payment.channelKey)
	}

	return
}

func (h *escrowPaymentHandler) CompleteAfterError(_payment handler.Payment, result error) (err *status.Status) {
	return
}
