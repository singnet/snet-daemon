package cmd

import (
	"fmt"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/spf13/cobra"
	"math/big"
)

// ListChannelsCmd shows list of channels from shared storage
var ChannelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage operations on payment channels",
	Long: "allows us to perform operations on channels with given channel ID." +
		" User can use 'snetd channel --unlock {channelID}' command to unlock the channel manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newChannelCommand)
	},
}

//Channel command type
type channelCommand struct {
	etcdclient       *etcddb.EtcdClient
	paymentChannelId *big.Int
}

// initializes and returns the new channel command object
func newChannelCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	channelId, err := getPaymentChannelId(cmd)
	if err != nil {
		return
	}
	command = &channelCommand{
		etcdclient:       components.EtcdClient(),
		paymentChannelId: channelId,
	}
	return
}

func getPaymentChannelId(cmd *cobra.Command) (id *big.Int, err error) {
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

// command's run method
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
	channelKey := "/payment-channel/lock/" + key.String()
	// verify whether the key exists or not
	_, ok, _ := command.etcdclient.Get(channelKey)
	if !ok {
		fmt.Println("Error: Channel is not found by key:", channelKey)
	}
	// if exists, delete the key
	return command.etcdclient.Delete(channelKey)
}
