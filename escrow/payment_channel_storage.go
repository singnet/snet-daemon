package escrow

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
)

// PaymentChannelStorage is an interface to get channel information by channel
// id.
type PaymentChannelStorage interface {
	// Get returns channel information by channel id. ok value indicates
	// whether passed key was found. err indicates storage error.
	Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error)
	// Put writes channel information by channel id.
	Put(key *PaymentChannelKey, state *PaymentChannelData) (err error)
	// Put writes channel information by channel id but only when key is
	// absent. ok is true if key was absent.
	PutIfAbsent(key *PaymentChannelKey, state *PaymentChannelData) (ok bool, err error)
	// CompareAndSwap atomically replaces old payment channel state by new
	// state. If ok flag is true and err is nil then operation was successful.
	// If err is nil and ok is false then operation failed because prevState is
	// not equal to current state. err indicates storage error.
	CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error)
}

type paymentChannelStorageImpl struct {
	delegate TypedAtomicStorage
}

// NewPaymentChannelStorage returns new instance of PaymentChannelStorage
// implementation
func NewPaymentChannelStorage(atomicStorage AtomicStorage) PaymentChannelStorage {
	return &paymentChannelStorageImpl{
		delegate: &TypedAtomicStorageImpl{
			atomicStorage: &PrefixedAtomicStorage{
				delegate:  atomicStorage,
				keyPrefix: "open-payment-",
			},
			keySerializer:     serialize,
			valueSerializer:   serialize,
			valueDeserializer: deserialize,
		},
	}
}

func serialize(value interface{}) (slice string, err error) {

	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	err = e.Encode(value)
	if err != nil {
		return
	}

	slice = string(b.Bytes())
	return
}

func deserialize(slice string, value interface{}) (err error) {

	b := bytes.NewBuffer([]byte(slice))
	d := gob.NewDecoder(b)
	err = d.Decode(value)
	return
}

func (storage *paymentChannelStorageImpl) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	result := &PaymentChannelData{}
	ok, err = storage.delegate.Get(key, result)
	if err != nil || !ok {
		return nil, ok, err
	}
	return result, ok, err
}

func (storage *paymentChannelStorageImpl) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *paymentChannelStorageImpl) PutIfAbsent(key *PaymentChannelKey, state *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *paymentChannelStorageImpl) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

// BlockchainChannelReader reads channel state from blockchain
type BlockchainChannelReader struct {
	replicaGroupID            func() (*big.Int, error)
	readChannelFromBlockchain func(channelID *big.Int) (channel *blockchain.MultiPartyEscrowChannel, ok bool, err error)
}

// NewBlockchainChannelReader returns new instance of blockchain channel reader
func NewBlockchainChannelReader(processor *blockchain.Processor, cfg *viper.Viper) *BlockchainChannelReader {
	return &BlockchainChannelReader{
		replicaGroupID: func() (*big.Int, error) {
			return config.GetBigIntFromViper(cfg, config.ReplicaGroupIDKey)
		},
		readChannelFromBlockchain: processor.MultiPartyEscrowChannel,
	}
}

// GetChannelStateFromBlockchain returns channel state from Ethereum
// blockchain. ok is false if channel was not found.
func (reader *BlockchainChannelReader) GetChannelStateFromBlockchain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	ch, ok, err := reader.readChannelFromBlockchain(key.ID)
	if err != nil || !ok {
		return
	}

	configGroupID, err := reader.replicaGroupID()
	if err != nil {
		return nil, false, err
	}
	if ch.GroupId.Cmp(configGroupID) != 0 {
		log.WithField("configGroupId", configGroupID).Warn("Channel received belongs to another group of replicas")
		return nil, false, fmt.Errorf("Channel received belongs to another group of replicas, current group: %v, channel group: %v", configGroupID, ch.GroupId)
	}

	// TODO: check recipient

	return &PaymentChannelData{
		Nonce:            ch.Nonce,
		State:            Open,
		Sender:           ch.Sender,
		Recipient:        ch.Recipient,
		GroupID:          ch.GroupId,
		FullAmount:       ch.Value,
		Expiration:       ch.Expiration,
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}, true, nil
}

// MergeStorageAndBlockchainChannelState merges two instances of payment
// channel: one read from storage, one from blockchain.
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
