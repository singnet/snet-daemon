package escrow

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"math/big"
	"reflect"

	"github.com/singnet/snet-daemon/blockchain"
)

// PaymentChannelStorage is a storage for PaymentChannelData by
// PaymentChannelKey based on TypedAtomicStorage implementation
type PaymentChannelStorage struct {
	delegate TypedAtomicStorage
}

// NewPaymentChannelStorage returns new instance of PaymentChannelStorage
// implementation
func NewPaymentChannelStorage(atomicStorage AtomicStorage,metadata *blockchain.ServiceMetadata) *PaymentChannelStorage {
	return &PaymentChannelStorage{
		delegate: &TypedAtomicStorageImpl{
			atomicStorage: &PrefixedAtomicStorage{
				delegate:  atomicStorage,
				//Add the MPE Network address as the prefix on the key for storage
				keyPrefix: "/"+metadata.MpeAddress+"/payment-channel/storage",
			},
			keySerializer:     serialize,
			valueSerializer:   serialize,
			valueDeserializer: deserialize,
			valueType:         reflect.TypeOf(PaymentChannelData{}),
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

// Get returns payment channel by key
func (storage *PaymentChannelStorage) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*PaymentChannelData), ok, err
}

// GetAll returns all channels from the storage
func (storage *PaymentChannelStorage) GetAll() (states []*PaymentChannelData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*PaymentChannelData), nil
}

// Put stores payment channel by key
func (storage *PaymentChannelStorage) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, state)
}

// PutIfAbsent storage payment channel by key if key is absent
func (storage *PaymentChannelStorage) PutIfAbsent(key *PaymentChannelKey, state *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

// CompareAndSwap compares previous storage value and set new value by key
func (storage *PaymentChannelStorage) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

// BlockchainChannelReader reads channel state from blockchain
type BlockchainChannelReader struct {

	readChannelFromBlockchain func(channelID *big.Int) (channel *blockchain.MultiPartyEscrowChannel, ok bool, err error)
	recipientPaymentAddress   func() common.Address
	readAllChannelsFromBlockChain func(sender *common.Address) (channelIds []*big.Int, ok bool, err error)
}

// NewBlockchainChannelReader returns new instance of blockchain channel reader
func NewBlockchainChannelReader(processor *blockchain.Processor, cfg *viper.Viper,
	orgMetadata *blockchain.OrganizationMetaData) *BlockchainChannelReader {
	return &BlockchainChannelReader{
		readChannelFromBlockchain: processor.MultiPartyEscrowChannel,
		recipientPaymentAddress: func() common.Address {
			address := orgMetadata.GetPaymentAddress()
			return address
		},
		readAllChannelsFromBlockChain:processor.BlockChainChannelIds,
	}
}

// GetChannelStateFromBlockchain returns channel state from Ethereum
// blockchain. ok is false if channel was not found.
func (reader *BlockchainChannelReader) GetChannelStateFromBlockchain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	ch, ok, err := reader.readChannelFromBlockchain(key.ID)
	if err != nil || !ok {
		return
	}


	recipientPaymentAddress := reader.recipientPaymentAddress()


	if recipientPaymentAddress != ch.Recipient {
		log.WithField("recipientPaymentAddress", recipientPaymentAddress).
			WithField("ch.Recipient", ch.Recipient).
			Warn("Recipient Address from service metadata not Match on what was retrieved from Channel")
		return nil, false, fmt.Errorf("recipient Address from service metadata does not Match on what was retrieved from Channel")
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
