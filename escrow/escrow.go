package escrow

import (
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
)

// paymentChannelService implements PaymentChannelService interface
type paymentChannelService struct {
	config     *viper.Viper
	storage    PaymentChannelStorage
	blockchain EscrowBlockchainApi
}

// NewPaymentChannelService returns instance of PaymentChannelService to work
// with payments via MultiPartyEscrow contract.
func NewPaymentChannelService(
	processor *blockchain.Processor,
	storage PaymentChannelStorage,
	config *viper.Viper) PaymentChannelService {

	return &paymentChannelService{
		config:     config,
		storage:    storage,
		blockchain: processor,
	}
}

func (h *paymentChannelService) PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
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

func (h *paymentChannelService) getChannelStateFromBlockchain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
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
	finish  func() error
}

func (claim *claimImpl) Payment() *Payment {
	return claim.payment
}

func (claim *claimImpl) Finish() error {
	return claim.finish()
}

func (h *paymentChannelService) StartClaim(key *PaymentChannelKey, update ChannelUpdate) (claim Claim, err error) {
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
		finish:  func() error { return nil },
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
	service *paymentChannelService
}

func (p *paymentTransaction) String() string {
	return fmt.Sprintf("{payment: %v, channel: %v}", p.payment, p.channel)
}

func (p *paymentTransaction) Channel() *PaymentChannelData {
	return p.channel
}

func (h *paymentChannelService) StartPaymentTransaction(payment *Payment) (transaction PaymentTransaction, err error) {
	channelKey := &PaymentChannelKey{ID: payment.ChannelID}
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
	}, nil
}

func (h *paymentChannelService) CurrentBlock() (currentBlock *big.Int, err error) {
	return h.blockchain.CurrentBlock()
}

func (h *paymentChannelService) PaymentExpirationThreshold() (threshold *big.Int) {
	return big.NewInt(h.config.GetInt64(config.PaymentExpirationThresholdBlocksKey))
}

func (payment *paymentTransaction) Commit() error {
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
