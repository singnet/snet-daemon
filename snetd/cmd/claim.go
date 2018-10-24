package cmd

import (
	"fmt"
	//"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/escrow"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
)

var ClaimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim money from payment channel",
	Long:  "Increment payment channel nonce and send blockchain transaction to claim money from channel",
	Run: func(cmd *cobra.Command, args []string) {
		err := runAndCleanup(cmd, args)
		if err != nil {
			log.Fatal(err.Error())
		}
	},
}

func runAndCleanup(cmd *cobra.Command, args []string) (err error) {
	components, err := InitComponents(cmd)
	if err != nil {
		return
	}
	defer components.Close()

	command, err := newClaimCommand(cmd, args, components)
	if err != nil {
		return
	}

	return command.Run()
}

type claimCommand struct {
	channelId *big.Int
	storage   escrow.PaymentChannelStorage
	channel   *escrow.PaymentChannelData
}

func newClaimCommand(cmd *cobra.Command, args []string, components *Components) (command *claimCommand, err error) {
	channelId, err := getChannelId(cmd)
	if err != nil {
		return
	}

	command = &claimCommand{
		channelId: channelId,
		storage:   components.PaymentChannelStorage(),
	}

	return
}

func getChannelId(cmd *cobra.Command) (id *big.Int, err error) {
	str := cmd.Flags().Lookup(ClaimChannelIdFlag).Value.String()
	value := &big.Int{}
	err = value.UnmarshalText([]byte(str))
	if err != nil {
		return nil, fmt.Errorf("Incorrect decimal number format: %v, error: %v", str, err)
	}
	return value, nil
}

func (command *claimCommand) Run() (err error) {
	err = command.getChannel()
	if err != nil {
		return
	}

	err = command.incrementChannelNonce()
	if err != nil {
		return
	}

	return
}

func (command *claimCommand) getChannel() (err error) {
	var ok bool
	command.channel, ok, err = command.storage.Get(&escrow.PaymentChannelKey{ID: command.channelId})
	if err != nil {
		return fmt.Errorf("Channel storage error: %v", err)
	}
	if !ok {
		return fmt.Errorf("Channel is not found, channel id: %v", command.channelId)
	}
	return nil
}

func (command *claimCommand) incrementChannelNonce() (err error) {
	nextChannel := *command.channel
	nextChannel.Nonce = (&big.Int{}).Add(nextChannel.Nonce, big.NewInt(1))

	ok, err := command.storage.CompareAndSwap(&escrow.PaymentChannelKey{ID: command.channelId}, command.channel, &nextChannel)
	if err != nil {
		return fmt.Errorf("Channel storage error: %v", err)
	}
	if !ok {
		return fmt.Errorf("Channel was concurrently updated, channel id: %v", command.channelId)
	}
	return nil
}
