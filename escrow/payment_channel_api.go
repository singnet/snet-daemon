package escrow

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/singnet/snet-daemon/blockchain"
)

// Payment contains MultiPartyEscrow payment details
type Payment struct {
	// MpeContractAddress is an address of the MultiPartyEscrow contract which
	// were used to open the payment channel.
	MpeContractAddress common.Address
	// ChannelID is an id of the payment channel used.
	ChannelID *big.Int
	// ChannelNonce is a nonce of the payment channel.
	ChannelNonce *big.Int
	// Amount is an amount of the payment.
	Amount *big.Int
	// Signature is a signature of the payment.
	Signature []byte
}

func (p *Payment) String() string {
	return fmt.Sprintf("{MpeContractAddress: %v, ChannelID: %v, ChannelNonce: %v, Amount: %v, Signature: %v}",
		p.MpeContractAddress, p.ChannelID, p.ChannelNonce, p.Amount, blockchain.BytesToBase64(p.Signature))
}

// PaymentChannelKey specifies the channel in MultiPartyEscrow contract. It
// consists of two parts: channel id and channel nonce. Channel nonce is
// incremented each time when amount of tokens in channel descreases. Nonce
// allows reusing channel id without risk of overexpenditure.
type PaymentChannelKey struct {
	ID *big.Int
}

func (key *PaymentChannelKey) String() string {
	return fmt.Sprintf("{ID: %v}", key.ID)
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

func (state PaymentChannelState) String() string {
	return [...]string{
		"Open",
		"Closed",
	}[state]
}

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
	// Expiration is a time at which channel will be expired. This time is
	// expressed in Ethereum block number. Since this block is added to
	// blockchain Sender can withdraw tokens from channel.
	Expiration *big.Int

	// service provider. This amount increments on price after each successful
	// RPC call.
	AuthorizedAmount *big.Int
	// Signature is a signature of last message containing Authorized amount.
	// It is required to claim tokens from channel.
	Signature []byte
}

func (data *PaymentChannelData) String() string {
	return fmt.Sprintf("{Nonce: %v, State: %v, Sender: %v, Recipient: %v, GroupId: %v, FullAmount: %v, Expiration: %v, AuthorizedAmount: %v, Signature: %v",
		data.Nonce, data.State, blockchain.AddressToHex(&data.Sender), blockchain.AddressToHex(&data.Recipient), data.GroupId, data.FullAmount, data.Expiration, data.AuthorizedAmount, blockchain.BytesToBase64(data.Signature))
}

// PaymentChannelService interface is API for payment channel functionality.
type PaymentChannelService interface {
	// PaymentChannel returns latest payment channel state. This method uses
	// shared storage and blockchain to construct and return latest channel
	// state.
	PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error)

	// StartClaim gets channel from storage, applies update on it and adds
	// payment for claiming into the storage.
	StartClaim(key *PaymentChannelKey, update ChannelUpdate) (claim Claim, err error)

	// StartPaymentTransaction validates payment and starts payment transaction
	StartPaymentTransaction(payment *Payment) (transaction PaymentTransaction, err error)
}

// PaymentErrorCode contains all types of errors which we need to handle on the
// client side.
type PaymentErrorCode int

const (
	// Internal error code means that error is caused by improper daemon
	// configuration or functioning. Client cannot do anything with it.
	Internal PaymentErrorCode = 1
	// Unauthenticated error code means that client sent payment which cannot
	// be applied to the channel.
	Unauthenticated PaymentErrorCode = 2
)

// PaymentError contains error code and message and implements Error interface.
type PaymentError struct {
	// Code is error code
	Code PaymentErrorCode
	// Message is message
	Message string
}

// NewPaymentError constructs new PaymentError instance with given error code
// and message.
func NewPaymentError(code PaymentErrorCode, format string, msg ...interface{}) *PaymentError {
	return &PaymentError{Code: code, Message: fmt.Sprintf(format, msg...)}
}

func (err *PaymentError) Error() string {
	return err.Message
}

// PaymentTransaction is a payment transaction in progress.
type PaymentTransaction interface {
	// Channel returns the channel which is used to apply the payment
	Channel() *PaymentChannelData
	// Commit finishes transaction and applies payment.
	Commit() error
	// Rollback rolls transaction back.
	Rollback() error
}

// Claim is a handle of payment channel claim in progress. It is returned by
// StartClaim method and provides caller information about payment to call
// MultiPartyEscrow.channelClaim function. After transaction is written to
// blockchain caller should call Finish() method to update payment repository
// state.
type Claim interface {
	// Payment returns the payment which is being claimed, caller uses details of
	// the payment to start blockchain transaction.
	Payment() *Payment
	// Finish to be called after blockchain transaction is finished successfully.
	// Updates repository state.
	Finish() error
}

// ChannelUpdate is an type of channel update which should be applied when
// StartClaim() method is called.
type ChannelUpdate func(channel *PaymentChannelData)

var (
	// CloseChannel is an update which zeroes full amount of the channel to
	// designate that channel sender should add funds to the channel before
	// continue working.
	CloseChannel ChannelUpdate = func(channel *PaymentChannelData) {
		channel.FullAmount = big.NewInt(0)
	}
	// IncrementChannelNonce is an update which increments channel nonce and
	// descreases full amount to allow channel sender continue working with
	// remaining amount.
	IncrementChannelNonce ChannelUpdate = func(channel *PaymentChannelData) {
		channel.Nonce = (&big.Int{}).Add(channel.Nonce, big.NewInt(1))
		channel.FullAmount = (&big.Int{}).Sub(channel.FullAmount, channel.AuthorizedAmount)
		channel.AuthorizedAmount = big.NewInt(0)
		channel.Signature = nil
	}
)

// EscrowBlockchainApi is an interface implemented by blockchain.Processor to
// provide blockchain operations related to MultiPartyEscrow contract
// processing.
type EscrowBlockchainApi interface {
	// EscrowContractAddress returns address of the MultiPartyEscrowContract
	EscrowContractAddress() common.Address
	// CurrentBlock returns current Ethereum blockchain block number
	CurrentBlock() (currentBlock *big.Int, err error)
	// MultiPartyEscrowChannel return MultiPartyEscrow channel by id
	MultiPartyEscrowChannel(channelID *big.Int) (channel *blockchain.MultiPartyEscrowChannel, ok bool, err error)
}
