package escrow

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
)

type combinedStorage struct {
	delegate PaymentChannelStorage
	mpe      *blockchain.MultiPartyEscrow
}

func NewCombinedStorage(processor *blockchain.Processor, delegate PaymentChannelStorage) PaymentChannelStorage {
	return &combinedStorage{
		delegate: delegate,
		mpe:      processor.MultiPartyEscrow(),
	}
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

	state, ok, err = storage.getChannelStateFromBlockchain(key.ID)
	if !ok || err != nil {
		return
	}
	log = log.WithField("state", state)
	log.Info("Channel found in blockchain")

	ok, err = storage.CompareAndSwap(key, nil, state)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		log.Warn("Key is already present in the storage")
		return nil, false, err
	}
	log.WithField("state", state).Info("Channel saved in storage")

	return
}

var zeroAddress = common.Address{}

func (storage *combinedStorage) getChannelStateFromBlockchain(id *big.Int) (state *PaymentChannelData, ok bool, err error) {
	log := log.WithField("id", id)

	channel, err := storage.mpe.Channels(nil, id)
	if err != nil {
		log.WithError(err).Warn("Error while looking up for channel id in blockchain")
		return nil, false, err
	}
	if channel.Sender == zeroAddress {
		log.Warn("Unable to find channel id in blockchain")
		return nil, false, nil
	}
	log = log.WithField("channel", channel)
	log.Debug("Channel found in blockchain")

	configGroupId := config.GetBigInt(config.ReplicaGroupIDKey)
	if channel.GroupId.Cmp(configGroupId) != 0 {
		log.WithField("configGroupId", configGroupId).Warn("Channel received belongs to another group of replicas")
		return nil, false, fmt.Errorf("Channel received belongs to another group of replicas, current group: %v, channel group: %v", configGroupId, channel.GroupId)
	}

	return &PaymentChannelData{
		Nonce:            channel.Nonce,
		State:            Open,
		Sender:           channel.Sender,
		Recipient:        channel.Recipient,
		GroupId:          channel.GroupId,
		FullAmount:       channel.Value,
		Expiration:       channel.Expiration,
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}, true, nil
}

func (storage *combinedStorage) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *combinedStorage) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}
