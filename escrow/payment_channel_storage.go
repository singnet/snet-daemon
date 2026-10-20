package escrow

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// PaymentChannelStorage stores PaymentChannelData under a channel-ID-only key.
// Each stored value contains the current nonce and local payment authorization.
type PaymentChannelStorage struct {
	delegate storage.TypedAtomicStorage
}

// NewPaymentChannelStorage creates typed channel storage under the
// /payment-channel/storage prefix.
func NewPaymentChannelStorage(atomicStorage storage.AtomicStorage) *PaymentChannelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/payment-channel/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeKey, reflect.TypeFor[PaymentChannelKey](), storage.EncodeGob, storage.DecodeGob,
		reflect.TypeFor[PaymentChannelData](),
	)
	return &PaymentChannelStorage{delegate: storage}

}

func serializeKey(key any) (slice string, err error) {
	return fmt.Sprintf("%v", key), nil
}

// Get returns a stored channel, with ok false if the key is absent.
func (storage *PaymentChannelStorage) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*PaymentChannelData), ok, err
}

// GetAll returns all stored channels without refreshing their blockchain state.
func (storage *PaymentChannelStorage) GetAll() (states []*PaymentChannelData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*PaymentChannelData), nil
}

// Put stores channel state, replacing any value under the same channel ID.
func (storage *PaymentChannelStorage) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, state)
}

// PutIfAbsent stores channel state only if no value exists under its key.
func (storage *PaymentChannelStorage) PutIfAbsent(key *PaymentChannelKey, state *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

// CompareAndSwap replaces the stored value only if it still matches prevState.
func (storage *PaymentChannelStorage) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

// BlockchainChannelReader reads channel state and checks the configured recipient.
type BlockchainChannelReader struct {
	readChannelFromBlockchain func(channelID *big.Int) (channel *blockchain.MultiPartyEscrowChannel, ok bool, err error)
	recipientPaymentAddress   func() common.Address
}

// NewBlockchainChannelReader uses the processor for channel reads and organization
// metadata for the expected recipient payment address.
func NewBlockchainChannelReader(processor blockchain.Processor, cfg *viper.Viper,
	orgMetadata *blockchain.OrganizationMetaData) *BlockchainChannelReader {
	return &BlockchainChannelReader{
		readChannelFromBlockchain: processor.MultiPartyEscrowChannel,
		recipientPaymentAddress: func() common.Address {
			address := orgMetadata.GetPaymentAddress()
			return address
		},
	}
}

// GetChannelStateFromBlockchain reads channel state and rejects a recipient that
// differs from the organization's payment address. Local authorizations are not
// loaded: AuthorizedAmount is zero and Signature is nil in the returned state.
func (reader *BlockchainChannelReader) GetChannelStateFromBlockchain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	ch, ok, err := reader.readChannelFromBlockchain(key.ID)
	if err != nil || !ok {
		zap.L().Warn("Unsuccessful GetChannelStateFromBlockchain", zap.Error(err), zap.Bool("ok", ok))
		return
	}

	recipientPaymentAddress := reader.recipientPaymentAddress()

	if recipientPaymentAddress != ch.Recipient {
		zap.L().Warn("Recipient Address from org metadata not Match on what was retrieved from Channel",
			zap.String("recipientPaymentAddress", recipientPaymentAddress.Hex()),
			zap.String("ch.Recipient", ch.Recipient.Hex()))
		return nil, false, fmt.Errorf("recipient Address from org metadata does not Match on what was retrieved from Channel")
	}
	return &PaymentChannelData{
		ChannelID:        key.ID,
		Nonce:            ch.Nonce,
		State:            Open,
		Sender:           ch.Sender,
		Recipient:        ch.Recipient,
		GroupID:          ch.GroupId,
		FullAmount:       ch.Value,
		Expiration:       ch.Expiration,
		Signer:           ch.Signer,
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}, true, nil
}

// MergeStorageAndBlockchainChannelState selects the state with the higher nonce.
// A higher local nonce represents a prepared claim that has not settled yet.
// At equal nonces, it preserves local authorizations and refreshes only FullAmount
// and Expiration from the blockchain. The input structs are not modified.
func MergeStorageAndBlockchainChannelState(storage, blockchain *PaymentChannelData) (merged *PaymentChannelData) {
	cmp := storage.Nonce.Cmp(blockchain.Nonce)
	if cmp > 0 {
		return storage
	}
	if cmp < 0 {
		return blockchain
	}

	tmp := *storage
	merged = &tmp
	merged.FullAmount = blockchain.FullAmount
	merged.Expiration = blockchain.Expiration

	return
}
