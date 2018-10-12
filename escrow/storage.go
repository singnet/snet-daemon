package escrow

import (
	"errors"
	"github.com/coreos/bbolt"
	"github.com/singnet/snet-daemon/blockchain"
	log "github.com/sirupsen/logrus"
)

type combinedStorage struct {
	delegate PaymentChannelStorage
}

func NewCombinedStorage(processor *blockchain.Processor, delegate PaymentChannelStorage) PaymentChannelStorage {
	return nil
}

func (storage *combinedStorage) Get(key *PaymentChannelKey) (state *PaymentChannelData, err error) {
	log := log.WithField("key", key)

	state, err = storage.delegate.Get(key)
	if err == nil {
		return
	}
	log.Info("Channel key is not found in storage")

	state, err = storage.getChannelStateFromBlockchain(key)
	if err != nil {
		return
	}
	log.WithField("state", state).Info("Channel found in blockchain")

	err = storage.CompareAndSwap(key, nil, state)
	if err != nil {
		log.Error("Cannot save channel in storage by key")
		return
	}
	log.WithField("state", state).Info("Channel saved in storage")

	return
}

func (storage *combinedStorage) getChannelStateFromBlockchain(key *PaymentChannelKey) (state *PaymentChannelData, err error) {
	// TODO: implement
	return nil, errors.New("not implemented yet")
}

func (storage *combinedStorage) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *combinedStorage) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

func NewDbStorage(db *bolt.DB) (storage PaymentChannelStorage) {
	// TODO: implement
	return nil
}
