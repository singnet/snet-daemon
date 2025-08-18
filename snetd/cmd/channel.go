package cmd

import (
	"fmt"
	"math/big"

	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/cobra"
)

// ChannelCmd manage operations on payment channels
var ChannelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage operations on payment channels",
	Long: "allows us to perform operations on channels with given channel ID." +
		" User can use 'snetd channel --unlock {channelID}' command to unlock the channel manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newChannelCommand)
	},
}

// Channel command type
type channelCommand struct {
	storage          storage.PrefixedAtomicStorage
	paymentChannelId *big.Int
}

// initializes and returns the new channel command object
func newChannelCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	channelId, err := getPaymentChannelId(cmd)
	if err != nil {
		return
	}
	command = &channelCommand{
		storage:          *components.LockerStorage(),
		paymentChannelId: channelId,
	}
	return
}

func getPaymentChannelId(*cobra.Command) (id *big.Int, err error) {
	if paymentChannelId == "" {
		return nil, nil
	}
	value := &big.Int{}
	err = value.UnmarshalText([]byte(paymentChannelId))
	if err != nil {
		return nil, fmt.Errorf("Incorrect decimal number format: %v, error: %v", paymentChannelId, err)
	}
	return value, nil
}

func (command *channelCommand) Run() (err error) {
	if command.paymentChannelId == nil {
		return fmt.Errorf("--unlock channel-id must be set")
	}
	if command.paymentChannelId != nil {
		return command.unlockChannel()
	}
	return
}

// unlocks the channel with a given channel ID
func (command *channelCommand) unlockChannel() (err error) {
	key := &escrow.PaymentChannelKey{}
	key.ID = command.paymentChannelId
	// check whether the key exists or not
	_, ok, err := command.storage.Get(key.String())
	if !ok {
		fmt.Printf("Error: Channel %s not found\n", key.String())
		return
	}
	// try deleting the key
	err = command.storage.Delete(key.String())
	if err != nil {
		fmt.Printf("Error: Unable to unlock the channel -%s\n", key.String())
		return
	}
	fmt.Printf("Success: Channel %s unlocked\n", key.String())
	return
}
