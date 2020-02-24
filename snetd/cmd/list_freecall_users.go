package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/singnet/snet-daemon/escrow"
)

// ListClaimsCmd shows list of channels from shared storage
var ListFreeCallUserCmd = &cobra.Command{
	Use:   "freecall-users",
	Short: "List all free call users",
	Long: "List all users who have availed free calls to your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newListFreeCallUserCommand)
	},
}

type listFreeCallUsersCommand struct {
	freeCallService escrow.FreeCallUserService
}

func newListFreeCallUserCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	command = &listFreeCallUsersCommand{
		freeCallService: components.FreeCallUserService(),
	}

	return
}

func (command *listFreeCallUsersCommand) Run() (err error) {
	users, err := command.freeCallService.ListFreeCallUsers()
	if err != nil {
		return
	}

	if len(users) == 0 {
		fmt.Println("no users of free calls , yet in  storage")
	}

	for _, user := range users {

		fmt.Printf("%v\n",user.String())
	}

	return nil
}
