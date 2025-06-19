package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/singnet/snet-daemon/v6/escrow"
)

// ListClaimsCmd shows list of channels from shared storage
var ListClaimsCmd = &cobra.Command{
	Use:   "claims",
	Short: "List payments which are not written to blockchain yet",
	Long: "List payments which are in progress state and not written" +
		" to the blockchain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newListClaimsCommand)
	},
}

type listClaimsCommand struct {
	channelService escrow.PaymentChannelService
}

func newListClaimsCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	command = &listClaimsCommand{
		channelService: components.PaymentChannelService(),
	}

	return
}

func (command *listClaimsCommand) Run() (err error) {
	claims, err := command.channelService.ListClaims()
	if err != nil {
		return
	}

	if len(claims) == 0 {
		fmt.Println("no claims in shared storage")
	}

	for _, claim := range claims {
		payment := claim.Payment()
		fmt.Printf("%v: %v\n", payment.ID(), payment)
	}

	return nil
}
