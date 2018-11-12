package escrow

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

type escrowPaymentType struct {
	payment Payment
	channel *PaymentChannelData
	service *paymentChannelService
}

func (p *escrowPaymentType) String() string {
	return fmt.Sprintf("{payment: %v, channel: %v}", p.payment, p.channel)
}

func (p *escrowPaymentType) Channel() *PaymentChannelData {
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

	return &escrowPaymentType{
		payment: *payment,
		channel: channel,
	}, nil
}

type paymentValidationContext interface {
	CurrentBlock() (currentBlock *big.Int, err error)
	PaymentExpirationThreshold() (threshold *big.Int)
}

func (h *paymentChannelService) CurrentBlock() (currentBlock *big.Int, err error) {
	return h.blockchain.CurrentBlock()
}

func (h *paymentChannelService) PaymentExpirationThreshold() (threshold *big.Int) {
	return big.NewInt(h.config.GetInt64(config.PaymentExpirationThresholdBlocksKey))
}

func validatePaymentUsingChannelState(context paymentValidationContext, payment *Payment, channel *PaymentChannelData) (err error) {
	var log = log.WithField("payment", payment).WithField("channel", channel)

	if payment.ChannelNonce.Cmp(channel.Nonce) != 0 {
		log.Warn("Incorrect nonce is sent by client")
		return NewPaymentError(Unauthenticated, "incorrect payment channel nonce, latest: %v, sent: %v", channel.Nonce, payment.ChannelNonce)
	}

	signerAddress, err := getSignerAddressFromPayment(payment)
	if err != nil {
		return
	}

	if *signerAddress != channel.Sender {
		log.WithField("signerAddress", blockchain.AddressToHex(signerAddress)).Warn("Channel sender is not equal to payment signer")
		return NewPaymentError(Unauthenticated, "payment is not signed by channel sender")
	}

	currentBlock, e := context.CurrentBlock()
	if e != nil {
		return NewPaymentError(Internal, "cannot determine current block")
	}
	expirationThreshold := context.PaymentExpirationThreshold()
	currentBlockWithThreshold := new(big.Int).Add(currentBlock, expirationThreshold)
	if currentBlockWithThreshold.Cmp(channel.Expiration) >= 0 {
		log.WithField("currentBlock", currentBlock).WithField("expirationThreshold", expirationThreshold).Warn("Channel expiration time is after expiration threshold")
		return NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: %v, current block: %v, expiration threshold: %v", channel.Expiration, currentBlock, expirationThreshold)
	}

	if channel.FullAmount.Cmp(payment.Amount) < 0 {
		log.Warn("Not enough tokens on payment channel")
		return NewPaymentError(Unauthenticated, "not enough tokens on payment channel, channel amount: %v, payment amount: %v", channel.FullAmount, payment.Amount)
	}

	return
}

func getSignerAddressFromPayment(payment *Payment) (signer *common.Address, err error) {
	message := bytes.Join([][]byte{
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	signer, e := getSignerAddressFromMessage(message, payment.Signature)
	if e != nil {
		return nil, NewPaymentError(Unauthenticated, "payment signature is not valid")
	}

	return
}

func getSignerAddressFromMessage(message, signature []byte) (signer *common.Address, err error) {
	log := log.WithFields(log.Fields{
		"message":   blockchain.BytesToBase64(message),
		"signature": blockchain.BytesToBase64(signature),
	})

	messageHash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	log = log.WithField("messageHash", hex.EncodeToString(messageHash))

	v, _, _, e := blockchain.ParseSignature(signature)
	if e != nil {
		log.WithError(e).Warn("Error parsing signature")
		return nil, errors.New("incorrect signature length")
	}

	modifiedSignature := bytes.Join([][]byte{signature[0:64], {v % 27}}, nil)
	publicKey, e := crypto.SigToPub(messageHash, modifiedSignature)
	if e != nil {
		log.WithError(e).WithField("modifiedSignature", modifiedSignature).Warn("Incorrect signature")
		return nil, errors.New("incorrect signature data")
	}
	log = log.WithField("publicKey", publicKey)

	keyOwnerAddress := crypto.PubkeyToAddress(*publicKey)
	log.WithField("keyOwnerAddress", keyOwnerAddress).Debug("Message signature parsed")

	return &keyOwnerAddress, nil
}

func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func bytesToBigInt(bytes []byte) *big.Int {
	return (&big.Int{}).SetBytes(bytes)
}

func (payment *escrowPaymentType) Commit() error {
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

func (payment *escrowPaymentType) Rollback() error {
	return nil
}
