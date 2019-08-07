package escrow

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/authutils"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"math/big"

	"github.com/singnet/snet-daemon/blockchain"
)
const (
	PrefixInSignature = "__MPE_claim_message"
	//Agreed constant value
	FreeCallPrefixValue = "__prefix_free_trial"
)

type FreeCallPaymentValidator struct {
	currentBlock               func() (currentBlock *big.Int, err error)
	freeCallSigner common.Address
}

func NewFreeCallPaymentValidator (funcCurrentBlock func() (currentBlock *big.Int, err error)) *FreeCallPaymentValidator {
	return &FreeCallPaymentValidator{
		currentBlock:funcCurrentBlock,
		freeCallSigner: common.HexToAddress(config.GetString(config.FreeCallSignerAddress)),
	}

}

func (validator *FreeCallPaymentValidator) Validate (payment *FreeCallPayment) (err error) {

	signerAddress, err := validator.getSignerAddressForFreeCall(payment)
	if err != nil {
		return NewPaymentError(Unauthenticated, "payment signature is not valid")
	}
     if signerAddress != &validator.freeCallSigner {
		 return NewPaymentError(Unauthenticated, "payment signer is not valid")
	 }
	// todo Calls to Metering service to check for allowed calls will go here
	return nil

}
// ChannelPaymentValidator validates payment using payment channel state.
type ChannelPaymentValidator struct {
	currentBlock               func() (currentBlock *big.Int, err error)
	paymentExpirationThreshold func() (threshold *big.Int)
}

// NewChannelPaymentValidator returns new payment validator instance
func NewChannelPaymentValidator(processor *blockchain.Processor, cfg *viper.Viper, metadata *blockchain.OrganizationMetaData) *ChannelPaymentValidator {
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
	var log = log.WithField("payment", payment).WithField("channel", channel)

	if payment.ChannelNonce.Cmp(channel.Nonce) != 0 {
		log.Warn("Incorrect nonce is sent by client")
		return NewPaymentError(IncorrectNonce, "incorrect payment channel nonce, latest: %v, sent: %v", channel.Nonce, payment.ChannelNonce)
	}

	signerAddress, err := getSignerAddressFromPayment(payment)
	if err != nil {
		return NewPaymentError(Unauthenticated, "payment signature is not valid")
	}

	log = log.WithField("signerAddress", blockchain.AddressToHex(signerAddress))
	if *signerAddress != channel.Signer {
		log.WithField("signerAddress", blockchain.AddressToHex(signerAddress)).Warn("Channel signer is not equal to payment signer")
		return NewPaymentError(Unauthenticated, "payment is not signed by channel signer")
	}
	currentBlock, e := validator.currentBlock()
	if e != nil {
		return NewPaymentError(Internal, "cannot determine current block")
	}
	expirationThreshold := validator.paymentExpirationThreshold()
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


func (validator *FreeCallPaymentValidator) getSignerAddressForFreeCall(payment *FreeCallPayment) (signer *common.Address, err error) {
     //Check for the current block Number
	if err := authutils.CompareWithLatestBlockNumber(payment.CurrentBlockNumber); err != nil {
		return nil, err
	}

	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixValue),
		[]byte(payment.UserId),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		bigIntToBytes(payment.CurrentBlockNumber),
	}, nil)

	signer, err = authutils.GetSignerAddressFromMessage(message, payment.Signature)
	if err != nil {
		log.WithField("payment", payment).WithError(err).Error("Cannot get signer from payment")
		return nil, err
	}
	return signer, err
}



func getSignerAddressFromPayment(payment *Payment) (signer *common.Address, err error) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		payment.MpeContractAddress.Bytes(),
		bigIntToBytes(payment.ChannelID),
		bigIntToBytes(payment.ChannelNonce),
		bigIntToBytes(payment.Amount),
	}, nil)

	signer, err = authutils.GetSignerAddressFromMessage(message, payment.Signature)
	if err != nil {
		log.WithField("payment", payment).WithError(err).Error("Cannot get signer from payment")
		return nil, err
	}

	return signer, err
}


func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

func bytesToBigInt(bytes []byte) *big.Int {
	return (&big.Int{}).SetBytes(bytes)
}
