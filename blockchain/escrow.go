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
		signature,
		sendBack,
	)
	if err != nil {
		log.WithError(err).Error("Error submitting transaction to claim funds from channel")
		return fmt.Errorf("Error submitting transaction to claim funds from channel: %v", err)
	}

	log.Info("Transaction sent, waiting for %v till transaction is committed", timeout)
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
