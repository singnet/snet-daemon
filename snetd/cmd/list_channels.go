package cmd

import (
	"github.com/spf13/cobra"

	"github.com/singnet/snet-daemon/escrow"
)

// ListChannelsCmd shows list of channels from shared storage
var ListChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List payment channels from the shared storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newListChannelsCommand)
	},
}

type listChannelsCommand struct {
	channelService escrow.PaymentChannelService
}

func newListChannelsCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	command = &listChannelsCommand{
		channelService: components.PaymentChannelService(),
	}

	return
}

func (command *listChannelsCommand) Run() error {
	return nil
}
