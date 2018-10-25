package escrow

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
)

const (
	// PaymentChannelIDHeader is a MultiPartyEscrow contract payment channel
	// id. Value is a string containing a decimal number.
	PaymentChannelIDHeader = "snet-payment-channel-id"
	// PaymentChannelNonceHeader is a payment channel nonce value. Value is a
	// string containing a decimal number.
	PaymentChannelNonceHeader = "snet-payment-channel-nonce"
	// PaymentChannelAmountHeader is an amount of payment channel value
	// which server is authorized to withdraw after handling the RPC call.
	// Value is a string containing a decimal number.
	PaymentChannelAmountHeader = "snet-payment-channel-amount"
	// PaymentChannelSignatureHeader is a signature of the client to confirm
	// amount withdrawing authorization. Value is an array of bytes.
	PaymentChannelSignatureHeader = "snet-payment-channel-signature-bin"

	// EscrowPaymentType each call should have id and nonce of payment channel
	// in metadata.
	EscrowPaymentType = "escrow"
)

// PaymentChannelKey specifies the channel in MultiPartyEscrow contract. It
// consists of two parts: channel id and channel nonce. Channel nonce is
// incremented each time when amount of tokens in channel descreases. Nonce
// allows reusing channel id without risk of overexpenditure.
type PaymentChannelKey struct {
	ID *big.Int
}

// PaymentChannelState is a current state of a payment channel. Payment
// channel may be in Open or Closed state.
type PaymentChannelState int

const (
	// Open means that channel is open and can be used to pay for calls.
	Open PaymentChannelState = 0
	// Closed means that channel is closed cannot be used to pay for calls.
	Closed PaymentChannelState = 1
)

// PaymentChannelData is to keep all channel related information.
type PaymentChannelData struct {
	// Nonce is a nonce of this channel state
	Nonce *big.Int
	// State is a payment channel state: Open or Closed.
	State PaymentChannelState
	// Sender is an Ethereum address of the client which created the channel.
	// It is and address to be charged for RPC call.
	Sender common.Address
	// Recipient is an address which can claim funds from channel using
	// signature. It is an address of service provider.
	Recipient common.Address
	// GroupId is an id of the group of service replicas which share the same
	// payment channel.
	GroupId *big.Int
	// FullAmount is an amount which is deposited in channel by Sender.
	FullAmount *big.Int
	// Expiration is a time at which channel will be expired. This time is
	// expressed in Ethereum block number. Since this block is added to
	// blockchain Sender can withdraw tokens from channel.
	Expiration *big.Int
	// AuthorizedAmount is current amount which Sender authorized to withdraw by
	// service provider. This amount increments on price after each successful
	// RPC call.
	AuthorizedAmount *big.Int
	// Signature is a signature of last message containing Authorized amount.
	// It is required to claim tokens from channel.
	Signature []byte
}

// PaymentChannelStorage is an interface to get channel information by channel
// id.
type PaymentChannelStorage interface {
	// Get returns channel information by channel id. ok value indicates
	// whether passed key was found. err indicates storage error.
	Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error)
	// Put writes channel information by channel id.
	Put(key *PaymentChannelKey, state *PaymentChannelData) (err error)
	// CompareAndSwap atomically replaces old payment channel state by new
	// state. If ok flag is true and err is nil then operation was successful.
	// If err is nil and ok is false then operation failed because prevState is
	// not equal to current state. err indicates storage error.
	CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error)
}

func (key PaymentChannelKey) String() string {
	return fmt.Sprintf("{ID: %v}", key.ID)
}

func (state PaymentChannelState) String() string {
	return [...]string{
		"Open",
		"Closed",
	}[state]
}

func (data PaymentChannelData) String() string {
	return fmt.Sprintf("{Nonce: %v. State: %v, Sender: %v, Recipient: %v, GroupId: %v, FullAmount: %v, Expiration: %v, AuthorizedAmount: %v, Signature: %v",
		data.Nonce, data.State, blockchain.AddressToHex(&data.Sender), blockchain.AddressToHex(&data.Recipient), data.GroupId, data.FullAmount, data.Expiration, data.AuthorizedAmount, blockchain.BytesToBase64(data.Signature))
}

type paymentChannelStorageImpl struct {
	AtomicStorage AtomicStorage
}

func NewPaymentChannelStorage(atomicStorage AtomicStorage) PaymentChannelStorage {
	return &paymentChannelStorageImpl{AtomicStorage: atomicStorage}
}

func (storage *paymentChannelStorageImpl) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {
	data, ok, err := storage.AtomicStorage.Get(key.String())
	if err != nil || !ok {
		return nil, ok, err
	}
	state = &PaymentChannelData{}
	err = deserialize([]byte(data), state)
	if err != nil {
		return nil, false, err
	}
	return state, true, nil
}

func (storage *paymentChannelStorageImpl) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	data, err := serialize(state)
	if err != nil {
		return
	}
	return storage.AtomicStorage.Put(key.String(), string(data))
}

func (storage *paymentChannelStorageImpl) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	newData, err := serialize(newState)
	if err != nil {
		return
	}

	if prevState == nil {
		return storage.AtomicStorage.PutIfAbsent(key.String(), string(newData))
	}

	prevData, err := serialize(prevState)
	if err != nil {
		return
	}

	return storage.AtomicStorage.CompareAndSwap(key.String(), string(prevData), string(newData))
}

// EscrowBlockchainApi is an interface implemented by blockchain.Processor to
// provide blockchain operations related to MultiPartyEscrow contract
// processing.
type EscrowBlockchainApi interface {
	// EscrowContractAddress returns address of the MultiPartyEscrowContract
	EscrowContractAddress() common.Address
	// CurrentBlock returns current Ethereum blockchain block number
	CurrentBlock() (currentBlock *big.Int, err error)
}

// escrowPaymentHandler implements paymentHandlerType interface
type escrowPaymentHandler struct {
	storage         PaymentChannelStorage
	incomeValidator IncomeValidator
	blockchain      EscrowBlockchainApi
}

// NewEscrowPaymentHandler returns instance of handler.PaymentHandler to validate
// payments via MultiPartyEscrow contract.
func NewEscrowPaymentHandler(processor *blockchain.Processor, storage PaymentChannelStorage, incomeValidator IncomeValidator) handler.PaymentHandler {
	return &escrowPaymentHandler{
		storage:         storage,
		incomeValidator: incomeValidator,
		blockchain:      processor,
	}
}

type escrowPaymentType struct {
	grpcContext  *handler.GrpcStreamContext
	channelID    *big.Int
	channelNonce *big.Int
	amount       *big.Int
	signature    []byte
	channel      *PaymentChannelData
}

func (p *escrowPaymentType) String() string {
	return fmt.Sprintf("{grpcContext: %v, channelID: %v, channelNonce: %v, amount: %v, signature: %v, channel: %v}",
		p.grpcContext, p.channelID, p.channelNonce, p.amount, blockchain.BytesToBase64(p.signature), p.channel)
}

func (h *escrowPaymentHandler) Type() (typ string) {
	return EscrowPaymentType
}

func (h *escrowPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *status.Status) {
	channelID, err := handler.GetBigInt(context.MD, PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(context.MD, PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	channelKey := &PaymentChannelKey{ID: channelID}
	channel, ok, e := h.storage.Get(channelKey)
	if e != nil {
		return nil, status.Newf(codes.Internal, "payment channel storage error")
	}
	if !ok {
		log.Warn("Payment channel not found")
		return nil, status.Newf(codes.InvalidArgument, "payment channel \"%v\" not found", channelKey)
	}

	amount, err := handler.GetBigInt(context.MD, PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(context.MD, PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &escrowPaymentType{
		grpcContext:  context,
		channelID:    channelID,
		channelNonce: channelNonce,
		amount:       amount,
		signature:    signature,
		channel:      channel,
	}, nil
}

func (h *escrowPaymentHandler) Validate(_payment handler.Payment) (err *status.Status) {
	var payment = _payment.(*escrowPaymentType)
	var log = log.WithField("payment", payment)

	if payment.channelNonce.Cmp(payment.channel.Nonce) != 0 {
		log.Warn("Incorrect nonce is sent by client")
		return status.Newf(codes.Unauthenticated, "incorrect payment channel nonce, latest: %v, sent: %v", payment.channel.Nonce, payment.channelNonce)
	}

	signerAddress, err := h.getSignerAddressFromPayment(payment)
	if err != nil {
		return
	}

	if *signerAddress != payment.channel.Sender {
		log.WithField("signerAddress", blockchain.AddressToHex(signerAddress)).Warn("Channel sender is not equal to payment signer")
		return status.New(codes.Unauthenticated, "payment is not signed by channel sender")
	}

	currentBlock, e := h.blockchain.CurrentBlock()
	if e != nil {
		return status.Newf(codes.Internal, "cannot determine current block")
	}
	if currentBlock.Cmp(payment.channel.Expiration) >= 0 {
		log.WithField("currentBlock", currentBlock).Warn("Channel is expired")
		return status.Newf(codes.Unauthenticated, "payment channel is expired since \"%v\" block", payment.channel.Expiration)
	}

	if payment.channel.FullAmount.Cmp(payment.amount) < 0 {
		log.Warn("Not enough tokens on payment channel")
		return status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: %v, payment amount: %v", payment.channel.FullAmount, payment.amount)
	}

	income := big.NewInt(0)
	income.Sub(payment.amount, payment.channel.AuthorizedAmount)
	err = h.incomeValidator.Validate(&IncomeData{Income: income, GrpcContext: payment.grpcContext})
	if err != nil {
		return
	}

	return
}

func (h *escrowPaymentHandler) getSignerAddressFromPayment(payment *escrowPaymentType) (signer *common.Address, err *status.Status) {
	message := bytes.Join([][]byte{
		h.blockchain.EscrowContractAddress().Bytes(),
		bigIntToBytes(payment.channelID),
		bigIntToBytes(payment.channelNonce),
		bigIntToBytes(payment.amount),
	}, nil)

	signer, e := getSignerAddressFromMessage(message, payment.signature)
	if e != nil {
		return nil, status.New(codes.Unauthenticated, "payment signature is not valid")
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

func (h *escrowPaymentHandler) Complete(_payment handler.Payment) (err *status.Status) {
	var payment = _payment.(*escrowPaymentType)
	ok, e := h.storage.CompareAndSwap(
		&PaymentChannelKey{ID: payment.channelID},
		payment.channel,
		&PaymentChannelData{
			Nonce:            payment.channel.Nonce,
			State:            payment.channel.State,
			Sender:           payment.channel.Sender,
			Recipient:        payment.channel.Recipient,
			FullAmount:       payment.channel.FullAmount,
			Expiration:       payment.channel.Expiration,
			AuthorizedAmount: payment.amount,
			Signature:        payment.signature,
			GroupId:          payment.channel.GroupId,
		},
	)
	if e != nil {
		log.WithError(e).Error("Unable to store new payment channel state")
		return status.New(codes.Internal, "unable to store new payment channel state")
	}
	if !ok {
		log.WithField("payment", payment).Warn("Channel state was changed concurrently")
		return status.Newf(codes.Unauthenticated, "state of payment channel was concurrently updated, channel id: %v", payment.channelID)
	}

	return
}

func (h *escrowPaymentHandler) CompleteAfterError(_payment handler.Payment, result error) (err *status.Status) {
	return
}

func serialize(value interface{}) (slice []byte, err error) {

	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	err = e.Encode(value)
	if err != nil {
		return
	}

	slice = b.Bytes()
	return
}

func deserialize(slice []byte, value interface{}) (err error) {

	b := bytes.NewBuffer(slice)
	d := gob.NewDecoder(b)
	err = d.Decode(value)
	return
}
