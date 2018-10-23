package cmd

import (
	//"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/etcddb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
)

var ClaimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim money from payment channel",
	Long:  "Increment payment channel nonce and send blockchain transaction to claim money from channel",
	Run: func(cmd *cobra.Command, args []string) {
		etcdStorage, err := etcddb.NewEtcdClient()
		if err != nil {
			log.WithError(err).Fatal("Unable initialize etcd client")
		}

		command := claimCommand{
			channelId: &escrow.PaymentChannelKey{ID: big.NewInt(0)}, // TODO: get channelId from command line
			storage:   escrow.NewPaymentChannelStorage(etcdStorage),
		}

		command.Run()
	},
}

type claimCommand struct {
	channelId *escrow.PaymentChannelKey
	storage   escrow.PaymentChannelStorage
}

func (command *claimCommand) Run() {
	channel := command.getChannel()

	nextChannel := *channel
	nextChannel.Nonce = (&big.Int{}).Add(channel.Nonce, big.NewInt(1))

	command.updateChannel(channel, &nextChannel)
}

func (command *claimCommand) getChannel() (channel *escrow.PaymentChannelData) {
	channel, ok, err := command.storage.Get(command.channelId)
	if err != nil {
		log.WithError(err).Fatal("Channel storage error")
	}
	if !ok {
		log.WithField("channelId", command.channelId).Fatal("Channel is not found")
	}
	return channel
}

func (command *claimCommand) updateChannel(currentChannel, nextChannel *escrow.PaymentChannelData) {
	ok, err := command.storage.CompareAndSwap(command.channelId, currentChannel, nextChannel)
	if err != nil {
		log.WithError(err).Fatal("Channel storage error")
	}
	if !ok {
		log.WithField("channelId", command.channelId).Fatal("Channel was concurrently updated")
	}
}
