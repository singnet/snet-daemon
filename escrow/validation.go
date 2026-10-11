package escrow

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"
	"go.uber.org/zap"
)

const (
	PrefixInSignature = "__MPE_claim_message"
	//Agreed constant value
	AllowedUserPrefixSignature = "__authorized_user"
)

type AllowedUserPaymentValidator struct {
}

func (validator *AllowedUserPaymentValidator) Validate(payment *Payment) (err error) {
	_, err = getSignerAddressFromPayment(payment)
	return err
}

// ChannelPaymentValidator validates payment using payment channel state.
type ChannelPaymentValidator struct {
	currentBlock               func() (currentBlock *big.Int, err error)
	paymentExpirationThreshold func() (threshold *big.Int)
}

// NewChannelPaymentValidator returns a new payment validator instance
func NewChannelPaymentValidator(processor blockchain.Processor, metadata *blockchain.OrganizationMetaData) *ChannelPaymentValidator {
	return &ChannelPaymentValidator{
		currentBlock: processor.CurrentBlock,
		paymentExpirationThreshold: func() *big.Int {
			return metadata.GetPaymentExpirationThreshold()
		},
	}
}

// Validate returns instance of PaymentError as error if validation fails, nil
// otherwise.
func (validator *ChannelPaymentValidator) Validate(payment *Payment, channel *PaymentChannelData) (err error) {
	paymentFieldLog := zap.Any("payment", payment)
	channelFieldLog := zap.Any("channel", channel)

	if payment.ChannelNonce.Cmp(channel.Nonce) != 0 {
		zap.L().Warn("Incorrect nonce is sent by client", paymentFieldLog, channelFieldLog)
		return NewPaymentError(IncorrectNonce, "incorrect payment channel nonce, latest: %v, sent: %v", channel.Nonce, payment.ChannelNonce)
	}

	signerAddress, err := getSignerAddressFromPayment(payment)
	if err != nil {
		return NewPaymentError(Unauthenticated, "payment signature is not valid")
	}

	signerAddressFieldLog := zap.String("signerAddress", utils.AddressToHex(signerAddress))
	if *signerAddress != channel.Signer && *signerAddress != channel.Sender {
		zap.L().Warn("Channel signer is not equal to payment signer/sender", signerAddressFieldLog)
		return NewPaymentError(Unauthenticated, "payment is not signed by channel signer/sender")
	}
	currentBlock, e := validator.currentBlock()
	if e != nil {
		return NewPaymentError(Internal, "cannot determine current block")
	}
	expirationThreshold := validator.paymentExpirationThreshold()
	currentBlockWithThreshold := new(big.Int).Add(currentBlock, expirationThreshold)
	if currentBlockWithThreshold.Cmp(channel.Expiration) >= 0 {
		zap.L().Warn("Channel expiration time is after expiration threshold", zap.Any("currentBlock", currentBlock), zap.Any("expirationThreshold", expirationThreshold))
		return NewPaymentError(Unauthenticated, "payment channel is near to be expired, expiration time: %v, current block: %v, expiration threshold: %v", channel.Expiration, currentBlock, expirationThreshold)
	}

	if channel.FullAmount.Cmp(payment.Amount) < 0 {
		zap.L().Warn("Not enough tokens on payment channel")
		return NewPaymentError(Unauthenticated, "not enough tokens on payment channel, channel amount: %v, payment amount: %v", channel.FullAmount, payment.Amount)
	}

	return
}

func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func bytesToBigInt(bytes []byte) *big.Int {
	return (&big.Int{}).SetBytes(bytes)
}

func checkCurationValidations(signer *common.Address) error {
	//This is only to protect the Service provider in test environment from being
	//hit by unknown users during a curation process
	if config.GetBool(config.AllowedUserFlag) {
		if !config.IsAllowedUser(signer) {
			return fmt.Errorf("you are not Authorized to call this service during curation process")
		}
	}
	return nil
}

func getSignerAddressFromPayment(payment *Payment) (signer *common.Address, err error) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	signer, err = utils.GetSignerAddressFromMessage(message, payment.Signature)
	if err != nil {
		zap.L().Error("Cannot get signer from payment", zap.Error(err), zap.Any("payment", payment))
		return nil, err
	}
	if err = checkCurationValidations(signer); err != nil {
		zap.L().Error(err.Error())
		return nil, err
	}

	return signer, err
}
