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
		return RunAndCleanup(cmd, args, newListChannelsCommand)
	},
}

//Channel command type
type channelCommand struct {
	etcdclient       *etcddb.EtcdClient
	paymentChannelId *big.Int
}

// initializes and returns the new channel command object
func newChannelCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	channelId, err := getChannelId(cmd)
	if err != nil {
		return
	}
	if err != nil {
		return
	}

	command = &channelCommand{
		etcdclient:       components.etcdClient,
		paymentChannelId: channelId,
	}
	return
}

// command's run method
func (command *channelCommand) Run() (err error) {
	if command.paymentChannelId == nil {
		return fmt.Errorf("--channel-id must be set")
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
	return command.etcdclient.Delete(channelKey)
}
