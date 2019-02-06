package escrow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// lockingPaymentChannelService implements PaymentChannelService interface
// using locks around proxied service call to guarantee that only one payment
// at time is applied to channel
type lockingPaymentChannelService struct {
	storage          *PaymentChannelStorage
	paymentStorage   *PaymentStorage
	blockchainReader *BlockchainChannelReader
	locker           Locker
	validator        *ChannelPaymentValidator
}

// NewPaymentChannelService returns instance of PaymentChannelService to work
// with payments via MultiPartyEscrow contract.
func NewPaymentChannelService(
	storage *PaymentChannelStorage,
	paymentStorage *PaymentStorage,
	blockchainReader *BlockchainChannelReader,
	locker Locker,
	channelPaymentValidator *ChannelPaymentValidator) PaymentChannelService {

	return &lockingPaymentChannelService{
		storage:          storage,
		paymentStorage:   paymentStorage,
		blockchainReader: blockchainReader,
		locker:           locker,
		validator:        channelPaymentValidator,
	}
}

func (h *lockingPaymentChannelService) PaymentChannelFromBlockChain(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	return h.blockchainReader.GetChannelStateFromBlockchain(key)
}

func (h *lockingPaymentChannelService) PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	storageChannel, storageOk, err := h.storage.Get(key)
	if err != nil {
		return
	}

	blockchainChannel, blockchainOk, err := h.blockchainReader.GetChannelStateFromBlockchain(key)
	if !storageOk {
		return blockchainChannel, blockchainOk, err
	}
	if err != nil || !blockchainOk {
		return storageChannel, storageOk, nil
	}

	return MergeStorageAndBlockchainChannelState(storageChannel, blockchainChannel), true, nil
}

func (h *lockingPaymentChannelService) ListChannels() (channels []*PaymentChannelData, err error) {
	return h.storage.GetAll()
}

type claimImpl struct {
	paymentStorage *PaymentStorage
	payment        *Payment
}

func (claim *claimImpl) Payment() *Payment {
	return claim.payment
}

func (claim *claimImpl) Finish() (err error) {
	return claim.paymentStorage.Delete(claim.payment)
}

func (h *lockingPaymentChannelService) StartClaim(key *PaymentChannelKey, update ChannelUpdate) (claim Claim, err error) {
	lock, ok, err := h.locker.Lock(key.String())
	if err != nil {
		return nil, fmt.Errorf("cannot get mutex for channel: %v", key)
	}
	if !ok {
		return nil, fmt.Errorf("another transaction on channel: %v is in progress", key)
	}
	defer func() {
		e := lock.Unlock()
		if e != nil {
			log.WithError(e).WithField("key", key).WithField("err", err).Error("Transaction is cancelled because of err, but channel cannot be unlocked. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
		}
	}()

	channel, ok, err := h.storage.Get(key)
	if err != nil {
		return
	}
	if !ok {
		return nil, fmt.Errorf("Channel is not found by key: %v", key)
	}

	nextChannel := *channel
	update(&nextChannel)

	err = h.storage.Put(key, &nextChannel)
	if err != nil {
		return nil, fmt.Errorf("Channel storage error: %v", err)
	}

	payment := getPaymentFromChannel(channel)

	err = h.paymentStorage.Put(payment)
	if err != nil {
		log.WithField("payment", payment).Error("Cannot write payment into payment storage. Channel storage is already updated. Payment should be handled manually.")
		return
	}

	return &claimImpl{
		paymentStorage: h.paymentStorage,
		payment:        payment,
	}, nil
}

func (h *lockingPaymentChannelService) ListClaims() (claims []Claim, err error) {
	payments, err := h.paymentStorage.GetAll()
	if err != nil {
		return
	}

	claims = make([]Claim, 0, len(payments))
	for _, payment := range payments {
		claim := &claimImpl{
			paymentStorage: h.paymentStorage,
			payment:        payment,
		}
		claims = append(claims, claim)
	}

	return
}

func getPaymentFromChannel(channel *PaymentChannelData) *Payment {
	return &Payment{
		// TODO: add MpeContractAddress to channel state
		//MpeContractAddress: channel.MpeContractAddress,
		ChannelID:    channel.ChannelID,
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

func (payment *paymentTransaction) String() string {
	return fmt.Sprintf("{payment: %v, channel: %v}", payment.payment, payment.channel)
}

func (payment *paymentTransaction) Channel() *PaymentChannelData {
	return payment.channel
}

func (h *lockingPaymentChannelService) StartPaymentTransaction(payment *Payment) (transaction PaymentTransaction, err error) {
	channelKey := &PaymentChannelKey{ID: payment.ChannelID}

	lock, ok, err := h.locker.Lock(channelKey.String())
	if err != nil {
		return nil, NewPaymentError(Internal, "cannot get mutex for channel: %v", channelKey)
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
		return nil, NewPaymentError(Internal, "payment channel error:"+err.Error())
	}
	if !ok {
		log.Warn("Payment channel not found")
		return nil, NewPaymentError(Unauthenticated, "payment channel \"%v\" not found", channelKey)
	}

	err = h.validator.Validate(payment, channel)
	if err != nil {
		return
	}

	return &paymentTransaction{
		payment: *payment,
		channel: channel,
		lock:    lock,
		service: h,
	}, nil
}

func (payment *paymentTransaction) Commit() error {
	defer func(payment *paymentTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			log.WithError(err).WithField("payment", payment).Error("Channel cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
		} else {
			log.Debug("Channel unlocked")
		}
	}(payment)

	e := payment.service.storage.Put(
		&PaymentChannelKey{ID: payment.payment.ChannelID},
		&PaymentChannelData{
			ChannelID:        payment.channel.ChannelID,
			Nonce:            payment.channel.Nonce,
			State:            payment.channel.State,
			Sender:           payment.channel.Sender,
			Recipient:        payment.channel.Recipient,
			FullAmount:       payment.channel.FullAmount,
			Expiration:       payment.channel.Expiration,
			Signer:           payment.channel.Signer,
			AuthorizedAmount: payment.payment.Amount,
			Signature:        payment.payment.Signature,
			GroupID:          payment.channel.GroupID,
		},
	)
	if e != nil {
		log.WithError(e).Error("Unable to store new payment channel state")
		return NewPaymentError(Internal, "unable to store new payment channel state")
	}

	log.Debug("Payment completed")
	return nil
}

func (payment *paymentTransaction) Rollback() error {
	defer func(payment *paymentTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			log.WithError(err).WithField("payment", payment).Error("Channel cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
		} else {
			log.Debug("Payment rolled back, channel unlocked")
		}
	}(payment)
	return nil
}
