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

func (storage *combinedStorage) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	log := log.WithField("key", key)

	state, ok, err = storage.delegate.Get(key)
	if ok && err == nil {
		return
	}
	if err != nil {
		return nil, false, err
	}
	log.Info("Channel key is not found in storage")

	state, ok, err = storage.getChannelStateFromBlockchain(key)
	if !ok || err != nil {
		return
	}
	log.WithField("state", state).Info("Channel found in blockchain")

	ok, err = storage.CompareAndSwap(key, nil, state)
	if err != nil {
		return
	}
	if !ok {
		log.Warn("Key is already present in the storage")
		return nil, false, err
	}
	log.WithField("state", state).Info("Channel saved in storage")

	return
}

func (storage *combinedStorage) getChannelStateFromBlockchain(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	// TODO: implement
	return nil, false, errors.New("not implemented yet")
}

func (storage *combinedStorage) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *combinedStorage) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

func NewDbStorage(db *bolt.DB) (storage PaymentChannelStorage) {
	// TODO: implement
	return nil
}
