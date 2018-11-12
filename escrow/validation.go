package escrow

import (
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/singnet/snet-daemon/blockchain"
)

type paymentValidationContext interface {
	CurrentBlock() (currentBlock *big.Int, err error)
	PaymentExpirationThreshold() (threshold *big.Int)
}

func validatePaymentUsingChannelState(context paymentValidationContext, payment *Payment, channel *PaymentChannelData) (err error) {
	var log = log.WithField("payment", payment).WithField("channel", channel)

	if payment.ChannelNonce.Cmp(channel.Nonce) != 0 {
		log.Warn("Incorrect nonce is sent by client")
		return NewPaymentError(Unauthenticated, "incorrect payment channel nonce, latest: %v, sent: %v", channel.Nonce, payment.ChannelNonce)
	}

	signerAddress, err := getSignerAddressFromPayment(payment)
	if err != nil {
		return NewPaymentError(Unauthenticated, "payment signature is not valid")
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
