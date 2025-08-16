package escrow

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"go.uber.org/zap"
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
	replicaGroupID   func() [32]byte
}

// NewPaymentChannelService returns an instance of PaymentChannelService to work
// with payments via MultiPartyEscrow contract.
func NewPaymentChannelService(
	storage *PaymentChannelStorage,
	paymentStorage *PaymentStorage,
	blockchainReader *BlockchainChannelReader,
	locker Locker,
	channelPaymentValidator *ChannelPaymentValidator, groupIdReader func() [32]byte) PaymentChannelService {

	return &lockingPaymentChannelService{
		storage:          storage,
		paymentStorage:   paymentStorage,
		blockchainReader: blockchainReader,
		locker:           locker,
		validator:        channelPaymentValidator,
		replicaGroupID:   groupIdReader,
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
		// Group ID check is only done for the first time, when the channel is added to storage from the blockchain,
		// if the channel is already present in the storage, the group ID check is skipped.
		if blockchainChannel != nil {
			blockChainGroupID := h.replicaGroupID()
			if err = h.verifyGroupId(blockChainGroupID, blockchainChannel.GroupID); err != nil {
				return nil, false, err
			}
		}
		return blockchainChannel, blockchainOk, err
	}
	if err != nil || !blockchainOk {
		return storageChannel, storageOk, nil
	}

	return MergeStorageAndBlockchainChannelState(storageChannel, blockchainChannel), true, nil
}

// Check if the channel belongs to the same group ID
func (h *lockingPaymentChannelService) verifyGroupId(configGroupID [32]byte, blockChainGroupID [32]byte) error {
	if blockChainGroupID != configGroupID {
		zap.L().Warn("Channel received belongs to another group of replicas", zap.Any("configGroupId", configGroupID))
		return fmt.Errorf("channel received belongs to another group of replicas, current group: %v, channel group: %v", configGroupID, blockChainGroupID)
	}
	return nil
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
		zap.L().Error("StartClaim, unable to get lock!", zap.Any("PaymentChannelKey", key))
		return nil, fmt.Errorf("cannot get mutex for channel: %v because of %v", key, err)
	}
	if !ok {
		return nil, fmt.Errorf("another transaction on channel: %v is in progress", key)
	}
	defer func() {
		e := lock.Unlock()
		if e != nil {
			zap.L().Error("Transaction is cancelled because of err, but channel cannot be unlocked. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.",
				zap.Any("key", key),
				zap.Error(err))
		}
	}()

	channel, ok, err := h.storage.Get(key)
	if err != nil {
		zap.L().Error("StartClaim, unable to get channel from Storage!", zap.Any("channelKey", key))
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
		zap.L().Error("Cannot write payment into payment storage. Channel storage is already updated. Payment should be handled manually.",
			zap.Error(err),
			zap.Any("payment", payment))
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

func (payment *paymentTransaction) GetSender() common.Address {
	return payment.channel.Sender
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
		zap.L().Error("StartPaymentTransaction, unable to get lock!", zap.Error(err), zap.Any("channelKey", channelKey))
		return nil, NewPaymentError(Internal, "cannot get mutex for channel: %v", channelKey)
	}
	if !ok {
		return nil, NewPaymentError(FailedPrecondition, "another transaction on channel: %v is in progress", channelKey)
	}
	defer func(lock Lock) {
		if err != nil {
			e := lock.Unlock()
			if e != nil {
				zap.L().Error("Transaction is cancelled because of err, but channel cannot be unlocked. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.",
					zap.Error(err),
					zap.Any("channelKey", channelKey))
			}
		}
	}(lock)

	channel, ok, err := h.PaymentChannel(channelKey)
	if err != nil {
		zap.L().Error("StartPaymentTransaction, unable to get channel!", zap.Error(err), zap.Any("channelKey", channelKey))
		return nil, NewPaymentError(Internal, "payment channel error: %s", err.Error())
	}
	if !ok {
		zap.L().Warn("Payment channel not found")
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
			zap.L().Error("Channel cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.",
				zap.Error(err), zap.Any("payment", payment))
		} else {
			zap.L().Debug("Channel unlocked", zap.Int64("channelID", payment.channel.ChannelID.Int64()))
		}
	}(payment)

	err := payment.service.storage.Put(
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
	if err != nil {
		zap.L().Error("Unable to store new payment channel state", zap.Error(err))
		return NewPaymentError(Internal, "unable to store new payment channel state")
	}

	zap.L().Debug("Payment completed", zap.Uint64("channel.ChannelID", payment.channel.ChannelID.Uint64()), zap.Uint64("payment.ChannelID", payment.payment.ChannelID.Uint64()))
	return nil
}

func (payment *paymentTransaction) Rollback() error {
	defer func(payment *paymentTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			zap.L().Error("Channel cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.",
				zap.Error(err),
				zap.Any("payment", payment))
		} else {
			zap.L().Debug("Payment rolled back, channel unlocked")
		}
	}(payment)
	return nil
}
