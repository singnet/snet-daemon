package escrow

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/utils"
)

// Payment authorizes a cumulative amount to be claimed from a MultiPartyEscrow
// channel at a specific nonce. Amounts are expressed in token base units.
type Payment struct {
	// MpeContractAddress is an address of the MultiPartyEscrow contract which
	// was used to open the payment channel.
	MpeContractAddress common.Address
	// ChannelID is an id of the payment channel used.
	ChannelID *big.Int
	// ChannelNonce identifies the channel state covered by the signature.
	ChannelNonce *big.Int
	// Amount is the cumulative amount authorized for ChannelNonce, not the price
	// of a single call.
	Amount *big.Int
	// Signature authorizes the MPE address, channel ID, nonce, and amount using
	// the claim-message format checked by getSignerAddressFromPayment.
	Signature []byte
}

func (p *Payment) String() string {
	return fmt.Sprintf("{MpeContractAddress: %v, ChannelID: %v, ChannelNonce: %v, Amount: %v, Signature: %v}",
		utils.AddressToHex(&p.MpeContractAddress), p.ChannelID, p.ChannelNonce, p.Amount, utils.BytesToBase64(p.Signature))
}

func (p *Payment) ID() string {
	return PaymentID(p.ChannelID, p.ChannelNonce)
}

// PaymentID identifies a pending claim by channel ID and nonce. Unlike a channel
// storage key, it distinguishes successive claims for the same channel.
func PaymentID(channelID *big.Int, channelNonce *big.Int) string {
	return fmt.Sprintf("%v/%v", channelID, channelNonce)
}

// PaymentChannelKey identifies a channel by ID in channel storage and blockchain
// lookups. The nonce belongs to PaymentChannelData, not this key. Pending claims
// use PaymentID to distinguish authorizations for successive channel nonces.
type PaymentChannelKey struct {
	ID *big.Int
}

func (key *PaymentChannelKey) String() string {
	return fmt.Sprintf("{ID: %v}", key.ID)
}

// PaymentChannelState is the local Open or Closed marker stored with a channel.
type PaymentChannelState int

const (
	// Open marks an open channel; payment eligibility is validated separately.
	Open PaymentChannelState = 0
	// Closed marks a closed channel. Setting this marker alone does not close
	// the channel on the blockchain.
	Closed PaymentChannelState = 1
)

func (state PaymentChannelState) String() string {
	return [...]string{
		"Open",
		"Closed",
	}[state]
}

// PaymentChannelData combines blockchain channel details with locally accepted
// payment authorizations. A pending claim can advance the local nonce and reduce
// FullAmount before the corresponding blockchain transaction is submitted.
type PaymentChannelData struct {
	// ChannelID is the channel's identifier in the MultiPartyEscrow contract.
	ChannelID *big.Int
	// Nonce identifies this channel state and the signatures valid for it.
	Nonce *big.Int
	// State is a payment channel state: Open or Closed.
	State PaymentChannelState
	// Sender is the Ethereum address that funded the channel and can authorize
	// payments from it.
	Sender common.Address
	// Recipient is the provider's Ethereum address authorized to claim funds
	// using a payment signature.
	Recipient common.Address
	// GroupID is an id of the group of service replicas which share the same
	// payment channel.
	GroupID [32]byte
	// FullAmount is the channel balance for Nonce, before subtracting the
	// current AuthorizedAmount. Starting a claim deducts that authorization
	// locally when advancing to the next nonce.
	FullAmount *big.Int
	// Expiration is the blockchain block number at which the channel expires.
	// Payment validation also requires the configured expiration safety margin.
	Expiration *big.Int
	// Signer is the channel's designated payment-signing address. Payments from
	// Sender are also accepted by ChannelPaymentValidator.
	Signer common.Address

	// AuthorizedAmount is the latest accepted cumulative amount for Nonce.
	// Paid calls authorize their price as an increment; prepaid authorization
	// can cover future calls. Advancing the nonce resets this amount to zero.
	AuthorizedAmount *big.Int
	// Signature authorizes AuthorizedAmount for Nonce and is retained for a
	// future claim. It is cleared when the nonce advances.
	Signature []byte
}

func (data *PaymentChannelData) String() string {
	return fmt.Sprintf("{ChannelID: %v, Nonce: %v, State: %v, Sender: %v, Recipient: %v, GroupId: %v, FullAmount: %v, Expiration: %v, Signer: %v, AuthorizedAmount: %v, Signature: %v",
		data.ChannelID, data.Nonce, data.State, utils.AddressToHex(&data.Sender), utils.AddressToHex(&data.Recipient), utils.BytesToBase64(data.GroupID[:]), data.FullAmount, data.Expiration, utils.AddressToHex(&data.Signer), data.AuthorizedAmount, utils.BytesToBase64(data.Signature))
}

// PaymentChannelService manages local authorizations and prepares payment claims.
// Blockchain reads are used for validation; callers submit claim transactions.
type PaymentChannelService interface {
	// PaymentChannel merges stored and blockchain channel state. Stored state
	// is returned if the blockchain read fails or the channel is absent there.
	// Reading a previously unstored channel does not persist it.
	PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error)
	// ListChannels returns locally stored channels without refreshing them
	// from the blockchain.
	ListChannels() (channels []*PaymentChannelData, err error)

	// StartClaim locks a stored channel, persists update, and records the
	// pre-update authorization as a pending claim. These are separate writes:
	// an error recording the payment can leave the channel already updated.
	StartClaim(key *PaymentChannelKey, update ChannelUpdate) (claim Claim, err error)
	// ListClaims returns pending claims without checking blockchain settlement.
	ListClaims() (claim []Claim, err error)

	// StartPaymentTransaction locks the channel and validates the payment.
	// A successful transaction must be completed with Commit or Rollback.
	StartPaymentTransaction(payment *Payment) (transaction PaymentTransaction, err error)

	// PaymentChannelFromBlockChain reads channel details directly from the
	// blockchain without merging locally accepted authorizations.
	PaymentChannelFromBlockChain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error)
}

// PaymentErrorCode identifies errors returned by payment handlers.
type PaymentErrorCode = errs.PaymentErrorCode

const (
	Internal           = errs.Internal
	Unauthenticated    = errs.Unauthenticated
	FailedPrecondition = errs.FailedPrecondition
	IncorrectNonce     = errs.IncorrectNonce
)

// PaymentError is shared with other payment handlers.
type PaymentError = errs.PaymentError

// NewPaymentError constructs a payment error.
func NewPaymentError(code PaymentErrorCode, format string, args ...any) *PaymentError {
	return errs.NewPaymentError(code, format, args...)
}

// PaymentTransaction holds a channel lock until Commit or Rollback is called.
// Callers must invoke exactly one of these completion methods.
type PaymentTransaction interface {
	// Channel returns the channel snapshot used to validate this payment,
	// including the previously accepted AuthorizedAmount.
	Channel() *PaymentChannelData
	// Commit stores the new authorization and releases the channel lock.
	Commit() error
	// Rollback releases the channel lock without storing the new authorization.
	Rollback() error
}

// Claim holds a pending payment prepared for blockchain settlement. Starting a
// claim has already updated local channel state; Finish only removes the pending
// payment record after settlement has been confirmed.
type Claim interface {
	// Payment returns the pre-update nonce, amount, and signature needed to
	// submit the claim to the MultiPartyEscrow contract.
	Payment() *Payment
	// Finish deletes the pending payment record. It does not submit a
	// transaction or check settlement, so the caller must verify it first.
	Finish() error
}

// ChannelUpdate changes the local channel copy persisted by StartClaim.
type ChannelUpdate func(channel *PaymentChannelData)

var (
	// CloseChannel sets the local FullAmount to zero. It does not change State
	// or Nonce, or close the channel on the blockchain.
	CloseChannel ChannelUpdate = func(channel *PaymentChannelData) {
		channel.FullAmount = big.NewInt(0)
	}
	// IncrementChannelNonce prepares the next local channel state by deducting
	// AuthorizedAmount, incrementing Nonce, and clearing the authorization and
	// signature. The remaining balance can fund calls while the old claim settles.
	IncrementChannelNonce ChannelUpdate = func(channel *PaymentChannelData) {
		channel.Nonce = (&big.Int{}).Add(channel.Nonce, big.NewInt(1))
		channel.FullAmount = (&big.Int{}).Sub(channel.FullAmount, channel.AuthorizedAmount)
		channel.AuthorizedAmount = big.NewInt(0)
		channel.Signature = nil
	}
)
