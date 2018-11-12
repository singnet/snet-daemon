package escrow

import (
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
)

// lockingPaymentChannelService implements PaymentChannelService interface
// using locks around proxied service call to guarantee that only one payment
// at time is applied to channel
type lockingPaymentChannelService struct {
	config     *viper.Viper
	storage    PaymentChannelStorage
	blockchain EscrowBlockchainApi
	locker     Locker
}

// NewPaymentChannelService returns instance of PaymentChannelService to work
// with payments via MultiPartyEscrow contract.
func NewPaymentChannelService(
	processor *blockchain.Processor,
	storage PaymentChannelStorage,
	config *viper.Viper) PaymentChannelService {

	return &lockingPaymentChannelService{
		config:     config,
		storage:    storage,
		blockchain: processor,
	}
}

func (h *lockingPaymentChannelService) PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	storageChannel, storageOk, err := h.storage.Get(key)
	if err != nil {
		return
	}

	blockchainChannel, blockchainOk, err := h.getChannelStateFromBlockchain(key)
	if !storageOk {
		return blockchainChannel, blockchainOk, err
	}
	if err != nil || !blockchainOk {
		return storageChannel, storageOk, nil
	}

	return mergeStorageAndBlockchainChannelState(storageChannel, blockchainChannel), true, nil
}

func (h *lockingPaymentChannelService) getChannelStateFromBlockchain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	ch, ok, err := h.blockchain.MultiPartyEscrowChannel(key.ID)
	if err != nil || !ok {
		return
	}

	configGroupId, err := config.GetBigIntFromViper(h.config, config.ReplicaGroupIDKey)
	if err != nil {
		return nil, false, err
	}
	if ch.GroupId.Cmp(configGroupId) != 0 {
		log.WithField("configGroupId", configGroupId).Warn("Channel received belongs to another group of replicas")
		return nil, false, fmt.Errorf("Channel received belongs to another group of replicas, current group: %v, channel group: %v", configGroupId, ch.GroupId)
	}

	// TODO: check recipient

	return &PaymentChannelData{
		Nonce:            ch.Nonce,
		State:            Open,
		Sender:           ch.Sender,
		Recipient:        ch.Recipient,
		GroupId:          ch.GroupId,
		FullAmount:       ch.Value,
		Expiration:       ch.Expiration,
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}, true, nil
}

func mergeStorageAndBlockchainChannelState(storage, blockchain *PaymentChannelData) (merged *PaymentChannelData) {
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

type claimImpl struct {
	payment *Payment
	lock    Lock
}

func (claim *claimImpl) Payment() *Payment {
	return claim.payment
}

func (claim *claimImpl) Finish() (err error) {
	err = claim.lock.Unlock()
	if err != nil {
		log.WithError(err).WithField("claim.payment", claim.payment).Error("Channel cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
		return
	}
	return
}

func (h *lockingPaymentChannelService) StartClaim(key *PaymentChannelKey, update ChannelUpdate) (claim Claim, err error) {
	lock, ok, err := h.locker.Lock(key.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get mutex for channel: %v", key)
	}
	if !ok {
		return nil, fmt.Errorf("another transaction on channel: %v is in progress", key)
	}
	defer func(lock Lock) {
		if err != nil {
			e := lock.Unlock()
			if e != nil {
				log.WithError(e).WithField("key", key).WithField("err", err).Error("Transaction is cancelled because of err, but channel cannot be unlocked. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
			}
		}
	}(lock)

	channel, ok, err := h.storage.Get(key)
	if err != nil {
		return
	}
	if !ok {
		return nil, fmt.Errorf("Channel is not found by key: %v", key)
	}

	nextChannel := *channel
	update(&nextChannel)

	ok, err = h.storage.CompareAndSwap(key, channel, &nextChannel)
	if err != nil {
		return nil, fmt.Errorf("Channel storage error: %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("Channel was concurrently updated, channel key: %v", key)
	}

	return &claimImpl{
		payment: getPaymentFromChannel(key, channel),
	}, nil
}

func getPaymentFromChannel(key *PaymentChannelKey, channel *PaymentChannelData) *Payment {
	return &Payment{
		// TODO: add MpeContractAddress to channel state
		//MpeContractAddress: channel.MpeContractAddress,
		ChannelID:    key.ID,
		ChannelNonce: channel.Nonce,
		Amount:       channel.AuthorizedAmount,
		Signature:    channel.Signature,
	}
}

type paymentTransaction struct {
	payment Payment
	channel *PaymentChannelData
	service *lockingPaymentChannelService
	lock    Lock
}

func (p *paymentTransaction) String() string {
	return fmt.Sprintf("{payment: %v, channel: %v}", p.payment, p.channel)
}

func (p *paymentTransaction) Channel() *PaymentChannelData {
	return p.channel
}

func (h *lockingPaymentChannelService) StartPaymentTransaction(payment *Payment) (transaction PaymentTransaction, err error) {
	channelKey := &PaymentChannelKey{ID: payment.ChannelID}

	lock, ok, err := h.locker.Lock(channelKey.String())
	if err != nil {
		return nil, NewPaymentError(FailedPrecondition, "cannot get mutex for channel: %v", channelKey)
	}
	if !ok {
		return nil, NewPaymentError(FailedPrecondition, "another transaction on channel: %v is in progress", channelKey)
	}
	defer func(lock Lock) {
		if err != nil {
			e := lock.Unlock()
			if e != nil {
				log.WithError(e).WithField("channelKey", channelKey).WithField("err", err).Error("Transaction is cancelled because of err, but channel cannot be unlocked. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
			}
		}
	}(lock)

	channel, ok, err := h.PaymentChannel(channelKey)
	if err != nil {
		return nil, NewPaymentError(Internal, "payment channel storage error")
	}
	if !ok {
		log.Warn("Payment channel not found")
		return nil, NewPaymentError(Unauthenticated, "payment channel \"%v\" not found", channelKey)
	}

	err = validatePaymentUsingChannelState(h, payment, channel)
	if err != nil {
		return
	}

	return &paymentTransaction{
		payment: *payment,
		channel: channel,
		lock:    lock,
	}, nil
}

func (h *lockingPaymentChannelService) CurrentBlock() (currentBlock *big.Int, err error) {
	return h.blockchain.CurrentBlock()
}

func (h *lockingPaymentChannelService) PaymentExpirationThreshold() (threshold *big.Int) {
	return big.NewInt(h.config.GetInt64(config.PaymentExpirationThresholdBlocksKey))
}

func (payment *paymentTransaction) Commit() error {
	defer func(payment *paymentTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			log.WithError(err).WithField("payment", payment).Error("Channel cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
		}
	}(payment)
	ok, e := payment.service.storage.CompareAndSwap(
		&PaymentChannelKey{ID: payment.payment.ChannelID},
		payment.channel,
		&PaymentChannelData{
			Nonce:            payment.channel.Nonce,
			State:            payment.channel.State,
			Sender:           payment.channel.Sender,
			Recipient:        payment.channel.Recipient,
			FullAmount:       payment.channel.FullAmount,
			Expiration:       payment.channel.Expiration,
			AuthorizedAmount: payment.payment.Amount,
			Signature:        payment.payment.Signature,
			GroupId:          payment.channel.GroupId,
		},
	)
	if e != nil {
		log.WithError(e).Error("Unable to store new payment channel state")
		return NewPaymentError(Internal, "unable to store new payment channel state")
	}
	if !ok {
		log.WithField("payment", payment).Warn("Channel state was changed concurrently")
		return NewPaymentError(Unauthenticated, "state of payment channel was concurrently updated, channel id: %v", payment.payment.ChannelID)
	}

	return nil
}

func (payment *paymentTransaction) Rollback() error {
	return nil
}
