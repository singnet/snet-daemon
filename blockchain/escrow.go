package blockchain

import (
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

type PaymentChannelKey struct {
	Id    *big.Int
	Nonce *big.Int
}

type PaymentChannelState struct {
	FullAmount         *big.Int
	ExpirationDateTime time.Time
	AuthorizedAmount   *big.Int
	PaymentSignature   []byte
}

type PaymentChannelStorage interface {
	Get(key *PaymentChannelKey) *PaymentChannelState
	CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelState, newState *PaymentChannelState) error
}

// escrowPaymentHandler implements paymentHandlerType interface
type escrowPaymentHandler struct {
	md      metadata.MD
	storage PaymentChannelStorage
}

func newEscrowPaymentHandler() *escrowPaymentHandler {
	return &escrowPaymentHandler{}
}

func (h *escrowPaymentHandler) validatePayment() error {
	/*
		id, err := getBigInt(h.md, PaymentChannelIdHeader)
		if err != nil {
			return err
		}

		nonce, err := getBigInt(h.md, PaymentChannelNonceHeader)
		if err != nil {
			return err
		}

		paymentChannelState := h.storage.Get(&PaymentChannelKey{id, nonce})

		signature, err := getBytes(h.md, PaymentChannelSignatureHeader)
	*/

	return status.Errorf(codes.Unimplemented, "not implemented yet")
}

func (h *escrowPaymentHandler) completePayment() {
}

func (h *escrowPaymentHandler) completePaymentAfterError(err error) {
}
