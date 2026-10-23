package cmd

import (
	"fmt"
	"math/big"

	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/cobra"
)

// ChannelCmd exposes manual removal of a payment channel's shared-storage lock.
var ChannelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage operations on payment channels",
	Long: "allows us to perform operations on channels with given channel ID." +
		" User can use 'snetd channel --unlock {channelID}' command to unlock the channel manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newChannelCommand)
	},
}

// channelCommand operates on lock records, not on the MPE contract's channels.
type channelCommand struct {
	storage          storage.PrefixedAtomicStorage
	paymentChannelId *big.Int
}

// newChannelCommand selects the MPE-scoped lock store and parses the channel ID.
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
	return command.unlockChannel()
}

// unlockChannel deletes the channel's lock record. It does not close the MPE
// channel, transfer funds, or delete the stored channel/payment state.
func (command *channelCommand) unlockChannel() (err error) {
	key := &escrow.PaymentChannelKey{}
	key.ID = command.paymentChannelId
	// Check for the lock record, not for an on-chain channel.
	_, ok, err := command.storage.Get(key.String())
	if !ok {
		fmt.Printf("Error: Channel %s not found\n", key.String())
		return
	}
	// Remove only the lock key from the prefixed lock store.
	err = command.storage.Delete(key.String())
	if err != nil {
		fmt.Printf("Error: Unable to unlock the channel -%s\n", key.String())
		return
	}
	fmt.Printf("Success: Channel %s unlocked\n", key.String())
	return
}
