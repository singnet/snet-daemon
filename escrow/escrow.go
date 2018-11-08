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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
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

// EscrowBlockchainApi is an interface implemented by blockchain.Processor to
// provide blockchain operations related to MultiPartyEscrow contract
// processing.
type EscrowBlockchainApi interface {
	// EscrowContractAddress returns address of the MultiPartyEscrowContract
	EscrowContractAddress() common.Address
	// CurrentBlock returns current Ethereum blockchain block number
	CurrentBlock() (currentBlock *big.Int, err error)
}

// PaymentChannelService interface is API for payment channel functionality.
type PaymentChannelService interface {
	// PaymentChannel returns latest payment channel state. This method uses
	// shared storage and blockchain to construct and return latest channel
	// state.
	PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error)

	handler.PaymentHandler
}

// escrowPaymentHandler implements paymentHandlerType interface
type escrowPaymentHandler struct {
	config          *viper.Viper
	storage         PaymentChannelStorage
	incomeValidator IncomeValidator
	blockchain      EscrowBlockchainApi
	mpe             *blockchain.MultiPartyEscrow
}

// NewPaymentChannelService returns instance of handler.PaymentHandler to validate
// payments via MultiPartyEscrow contract.
func NewPaymentChannelService(
	processor *blockchain.Processor,
	storage PaymentChannelStorage,
	incomeValidator IncomeValidator,
	config *viper.Viper) PaymentChannelService {

	return &escrowPaymentHandler{
		config:          config,
		storage:         storage,
		incomeValidator: incomeValidator,
		blockchain:      processor,
		mpe:             processor.MultiPartyEscrow(),
	}
}

// Payment contains MultiPartyEscrow payment details
type Payment struct {
	// MpeContractAddress is an address of the MultiPartyEscrow contract which
	// were used to open the payment channel.
	MpeContractAddress common.Address
	// ChannelID is an id of the payment channel used.
	ChannelID *big.Int
	// ChannelNonce is a nonce of the payment channel.
	ChannelNonce *big.Int
	// Amount is an amount of the payment.
	Amount *big.Int
	// Signature is a signature of the payment.
	Signature []byte
}

type escrowPaymentType struct {
	payment Payment
	channel *PaymentChannelData
}

func (p *Payment) String() string {
	return fmt.Sprintf("{MpeContractAddress: %v, ChannelID: %v, ChannelNonce: %v, Amount: %v, Signature: %v}",
		p.MpeContractAddress, p.ChannelID, p.ChannelNonce, p.Amount, blockchain.BytesToBase64(p.Signature))
}

func (p *escrowPaymentType) String() string {
	return fmt.Sprintf("{payment: %v, channel: %v}", p.payment, p.channel)
}

func (h *escrowPaymentHandler) PaymentChannel(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	channel, ok, err = h.storage.Get(key)
	if err != nil {
		return
	}
	if ok {
		return
	}

	return getChannelStateFromBlockchain(h.mpe, key.ID)
}

func (h *escrowPaymentHandler) Type() (typ string) {
	return EscrowPaymentType
}

func (h *escrowPaymentHandler) Payment(context *handler.GrpcStreamContext) (payment handler.Payment, err *status.Status) {
	internalPayment, err := h.getPaymentFromContext(context)
	if err != nil {
		return
	}

	channelKey := &PaymentChannelKey{ID: internalPayment.ChannelID}
	channel, ok, e := h.storage.Get(channelKey)
	if e != nil {
		return nil, status.Newf(codes.Internal, "payment channel storage error")
	}
	if !ok {
		log.Warn("Payment channel not found")
		return nil, status.Newf(codes.InvalidArgument, "payment channel \"%v\" not found", channelKey)
	}

	err = validatePaymentUsingChannelState(h, internalPayment, channel)
	if err != nil {
		return nil, err
	}

	income := big.NewInt(0)
	income.Sub(internalPayment.Amount, channel.AuthorizedAmount)
	err = h.incomeValidator.Validate(&IncomeData{Income: income, GrpcContext: context})
	if err != nil {
		return
	}

	return &escrowPaymentType{
		payment: *internalPayment,
		channel: channel,
	}, nil
}

func (h *escrowPaymentHandler) getPaymentFromContext(context *handler.GrpcStreamContext) (payment *Payment, err *status.Status) {
	channelID, err := handler.GetBigInt(context.MD, PaymentChannelIDHeader)
	if err != nil {
		return
	}

	channelNonce, err := handler.GetBigInt(context.MD, PaymentChannelNonceHeader)
	if err != nil {
		return
	}

	amount, err := handler.GetBigInt(context.MD, PaymentChannelAmountHeader)
	if err != nil {
		return
	}

	signature, err := handler.GetBytes(context.MD, PaymentChannelSignatureHeader)
	if err != nil {
		return
	}

	return &Payment{
		MpeContractAddress: h.blockchain.EscrowContractAddress(),
		ChannelID:          channelID,
		ChannelNonce:       channelNonce,
		Amount:             amount,
		Signature:          signature,
	}, nil
}

func (h *escrowPaymentHandler) Validate(_payment handler.Payment) (err *status.Status) {
	return nil
}

type paymentValidationContext interface {
	CurrentBlock() (currentBlock *big.Int, err error)
	PaymentExpirationThreshold() (threshold *big.Int)
}

func (h *escrowPaymentHandler) CurrentBlock() (currentBlock *big.Int, err error) {
	return h.blockchain.CurrentBlock()
}

func (h *escrowPaymentHandler) PaymentExpirationThreshold() (threshold *big.Int) {
	return big.NewInt(h.config.GetInt64(config.PaymentExpirationThresholdBlocksKey))
}

func validatePaymentUsingChannelState(context paymentValidationContext, payment *Payment, channel *PaymentChannelData) (err *status.Status) {
	var log = log.WithField("payment", payment).WithField("channel", channel)

	if payment.ChannelNonce.Cmp(channel.Nonce) != 0 {
		log.Warn("Incorrect nonce is sent by client")
		return status.Newf(codes.Unauthenticated, "incorrect payment channel nonce, latest: %v, sent: %v", channel.Nonce, payment.ChannelNonce)
	}

	signerAddress, err := getSignerAddressFromPayment(payment)
	if err != nil {
		return
	}

	if *signerAddress != channel.Sender {
		log.WithField("signerAddress", blockchain.AddressToHex(signerAddress)).Warn("Channel sender is not equal to payment signer")
		return status.New(codes.Unauthenticated, "payment is not signed by channel sender")
	}

	currentBlock, e := context.CurrentBlock()
	if e != nil {
		return status.Newf(codes.Internal, "cannot determine current block")
	}
	expirationThreshold := context.PaymentExpirationThreshold()
	currentBlockWithThreshold := new(big.Int).Add(currentBlock, expirationThreshold)
	if currentBlockWithThreshold.Cmp(channel.Expiration) >= 0 {
		log.WithField("currentBlock", currentBlock).WithField("expirationThreshold", expirationThreshold).Warn("Channel expiration time is after expiration threshold")
		return status.Newf(codes.Unauthenticated, "payment channel is near to be expired, expiration time: %v, current block: %v, expiration threshold: %v", channel.Expiration, currentBlock, expirationThreshold)
	}

	if channel.FullAmount.Cmp(payment.Amount) < 0 {
		log.Warn("Not enough tokens on payment channel")
		return status.Newf(codes.Unauthenticated, "not enough tokens on payment channel, channel amount: %v, payment amount: %v", channel.FullAmount, payment.Amount)
	}

	return
}

func getSignerAddressFromPayment(payment *Payment) (signer *common.Address, err *status.Status) {
	message := bytes.Join([][]byte{
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	signer, e := getSignerAddressFromMessage(message, payment.Signature)
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
		return status.New(codes.Internal, "unable to store new payment channel state")
	}
	if !ok {
		log.WithField("payment", payment).Warn("Channel state was changed concurrently")
		return status.Newf(codes.Unauthenticated, "state of payment channel was concurrently updated, channel id: %v", payment.payment.ChannelID)
	}

	return
}

func (h *escrowPaymentHandler) CompleteAfterError(_payment handler.Payment, result error) (err *status.Status) {
	return
}
