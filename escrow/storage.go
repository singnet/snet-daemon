package escrow

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/singnet/snet-daemon/blockchain"
)

// PaymentChannelKey specifies the channel in MultiPartyEscrow contract. It
// consists of two parts: channel id and channel nonce. Channel nonce is
// incremented each time when amount of tokens in channel descreases. Nonce
// allows reusing channel id without risk of overexpenditure.
type PaymentChannelKey struct {
	ID *big.Int
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
	// Expiration is a time at which channel will be expired. This time is
	// expressed in Ethereum block number. Since this block is added to
	// blockchain Sender can withdraw tokens from channel.
	Expiration *big.Int
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
	return fmt.Sprintf("{ID: %v}", key.ID)
}

func (state PaymentChannelState) String() string {
	return [...]string{
		"Open",
		"Closed",
	}[state]
}

func (data PaymentChannelData) String() string {
	return fmt.Sprintf("{Nonce: %v. State: %v, Sender: %v, Recipient: %v, GroupId: %v, FullAmount: %v, Expiration: %v, AuthorizedAmount: %v, Signature: %v",
		data.Nonce, data.State, blockchain.AddressToHex(&data.Sender), blockchain.AddressToHex(&data.Recipient), data.GroupId, data.FullAmount, data.Expiration, data.AuthorizedAmount, blockchain.BytesToBase64(data.Signature))
}

type paymentChannelStorageImpl struct {
	AtomicStorage AtomicStorage
}

func NewPaymentChannelStorage(atomicStorage AtomicStorage) PaymentChannelStorage {
	return &paymentChannelStorageImpl{AtomicStorage: atomicStorage}
}

func (storage *paymentChannelStorageImpl) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	data, ok, err := storage.AtomicStorage.Get(key.String())
	if err != nil || !ok {
		return nil, ok, err
	}
	state = &PaymentChannelData{}
	err = deserialize([]byte(data), state)
	if err != nil {
		return nil, false, err
	}
	return state, true, nil
}

func (storage *paymentChannelStorageImpl) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	data, err := serialize(state)
	if err != nil {
		return
	}
	return storage.AtomicStorage.Put(key.String(), string(data))
}

func (storage *paymentChannelStorageImpl) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	newData, err := serialize(newState)
	if err != nil {
		return
	}

	if prevState == nil {
		return storage.AtomicStorage.PutIfAbsent(key.String(), string(newData))
	}

	prevData, err := serialize(prevState)
	if err != nil {
		return
	}

	return storage.AtomicStorage.CompareAndSwap(key.String(), string(prevData), string(newData))
}
