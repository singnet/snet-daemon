package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
)

func (processor *Processor) ClaimFundsFromChannel(timeout time.Duration, channelId, amount *big.Int, signature []byte, sendBack bool) (err error) {
	log := log.WithFields(log.Fields{
		"timeout":    timeout,
		"channelId":  channelId,
		"amount":     amount,
		"signature":  BytesToBase64(signature),
		"isSendBack": sendBack,
	})

	v, r, s, err := ParseSignature(signature)
	if err != nil {
		return
	}

	auth := bind.NewKeyedTransactor(processor.privateKey)

	log.Info("Submitting transaction to claim funds from channel")
	txn, err := processor.multiPartyEscrow.ChannelClaim(
		&bind.TransactOpts{
			From:     common.HexToAddress(processor.address),
			Signer:   auth.Signer,
			GasLimit: 1000000,
		},
		channelId,
		amount,
		v,
		r,
		s,
		sendBack,
	)
	if err != nil {
		log.WithError(err).Error("Error submitting transaction to claim funds from channel")
		return fmt.Errorf("Error submitting transaction to claim funds from channel: %v", err)
	}

	log.WithField("timeout", timeout).Info("Transaction sent, waiting for timeout till transaction is committed")
	endTime := time.Now().Add(timeout)
	isPending := true
	for {
		_, isPending, err = processor.ethClient.TransactionByHash(context.Background(), txn.Hash())
		if err != nil {
			log.WithError(err).Error("Transaction error")
			// TODO: fix properly
			//return fmt.Errorf("Error while committing blockchain transaction: %v", err)
		}
		if !isPending {
			break
		}
		if time.Now().After(endTime) {
			log.Error("Transaction timeout")
			return fmt.Errorf("Timeout while waiting for blockchain transaction commit")
		}
		time.Sleep(time.Second * 1)
	}

	log.Info("Transaction finished successfully")
	return nil
}

type MultiPartyEscrowChannel struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    [32]byte
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
}

var zeroAddress = common.Address{}

func (processor *Processor) MultiPartyEscrowChannel(channelID *big.Int) (channel *MultiPartyEscrowChannel, ok bool, err error) {
	log := log.WithField("channelID", channelID)

	ch, err := processor.multiPartyEscrow.Channels(nil, channelID)
	if err != nil {
		log.WithError(err).Warn("Error while looking up for channel id in blockchain")
		return nil, false, err
	}
	if ch.Sender == zeroAddress {
		log.Warn("Unable to find channel id in blockchain")
		return nil, false, nil
	}

	channel = &MultiPartyEscrowChannel{
		Sender:     ch.Sender,
		Recipient:  ch.Recipient,
		GroupId:    ch.GroupId,
		Value:      ch.Value,
		Nonce:      ch.Nonce,
		Expiration: ch.Expiration,
	}

	log = log.WithField("channel", channel)
	log.Debug("Channel found in blockchain")

	return channel, true, nil
}
