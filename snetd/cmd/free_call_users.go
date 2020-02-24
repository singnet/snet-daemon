package cmd

import (
	"fmt"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/spf13/cobra"

)

// ListChannelsCmd shows list of channels from shared storage
var FreeCallUserCmd = &cobra.Command{
	Use:   "freecall-users",
	Short: "Manage operations on Free Call Users",
	Long: "allows us to perform operations on free call users with given user_id(email_id)." +
		" User can use 'snetd freecalluser --unlock {userId}' command to unlock the User manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newfreeCallUserCommandCommand)
	},
}

//Channel command type
type freeCallUserCommand struct {
	storage   *escrow.PrefixedAtomicStorage
	userId    string
	component *Components
}

// initializes and returns the new channel command object
func newfreeCallUserCommandCommand(cmd *cobra.Command, args []string, pComponents *Components) (command Command, err error) {
	key, err := getUserlId(cmd)
	if err != nil {
		return
	}
	command = &freeCallUserCommand{
		storage:   pComponents.LockerStorage(),
		userId:    key,
		component: pComponents,
	}
	return
}

func getUserlId(cmd *cobra.Command) (userId string, err error) {
	//todo Incase we need to more validation on checking if the user ID is a valid email ID
	return freeCallUserId, nil
}

// command's run method
func (command *freeCallUserCommand) Run() (err error) {
	if command.userId == "" {
		return fmt.Errorf("--unlock user-id must be set")
	}
	return command.unlockFreeCallUser()
}

// release the lock on the user with the given user id
func (command *freeCallUserCommand) unlockFreeCallUser() (err error) {
	key := &escrow.FreeCallUserKey{}
	key.UserId = freeCallUserId
	key.OrganizationId = config.GetString(config.OrganizationId)
	key.ServiceId = config.GetString(config.ServiceId)
	key.GroupID = command.component.OrganizationMetaData().GetGroupIdString()
	// check whether the key exists or not
	_, ok, err := command.storage.Get(key.String())
	if !ok {
		fmt.Printf("Error: Free Call user %s not found\n", key.String())
		return
	}
	// try deleting the key
	err = command.storage.Delete(key.String())
	if err != nil {
		fmt.Printf("Error: Unable to unlock the user -%s\n", key.String())
		return
	}
	fmt.Printf("Success: User %s unlocked\n", key.String())
	return
}
