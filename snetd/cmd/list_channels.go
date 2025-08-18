package cmd

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/spf13/cobra"

	"github.com/singnet/snet-daemon/v6/escrow"
)

// ListChannelsCmd shows the list of channels from shared storage
var ListChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List payment channels",
	Long: "List payment channels for which at least on payment was received." +
		" User can use 'snetd claim --channel-id' command to claim funds from channel.",
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
	channels, err := command.channelService.ListChannels()
	if err != nil {
		zap.L().Error("can't list channels", zap.Error(err))
		return err
	}

	if len(channels) == 0 {
		fmt.Println("no channels in shared storage")
	}

	for _, channel := range channels {
		fmt.Printf("%v: %v\n", channel.ChannelID, channel)
	}

	return nil
}
