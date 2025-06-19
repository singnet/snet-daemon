package cmd

import (
	"fmt"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/cobra"
)

var FreeCallUserUnLockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock a free call user",
	Long: "allows us to perform operations on free call users with given user_id(email_id)." +
		" User can use 'snetd freecall unlock -u {userId}' command to unlock the User manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newFreeCallUserUnLockCommandCommand)
	},
}

var FreeCallUserResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the count on free calls used for the given user to zero",
	Long:  "User can use 'snetd freecall reset -u {user-id}' command to reset the free call used of this User manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newFreeCallResetCountCommand)
	},
}

// Free call user unlock command
type freeCallUserUnLockCommand struct {
	lockStorage *storage.PrefixedAtomicStorage
	userId      string
	address     string
	orgMetadata *blockchain.OrganizationMetaData
}

// Free call user unlock command
type freeCallUserResetCountCommand struct {
	lockStorage *storage.PrefixedAtomicStorage
	userStorage *escrow.FreeCallUserStorage
	userID      string
	address     string
	orgMetadata *blockchain.OrganizationMetaData
}

// initializes and returns the new unlock user of free calls command object
func newFreeCallUserUnLockCommandCommand(cmd *cobra.Command, args []string, pComponents *Components) (command Command, err error) {
	userID, address, err := getFreeCallIDs(cmd)
	if err != nil {
		return
	}
	command = &freeCallUserUnLockCommand{
		lockStorage: pComponents.FreeCallLockerStorage(),
		userId:      userID,
		address:     address,
		orgMetadata: pComponents.OrganizationMetaData(),
	}
	return
}

// initializes and returns the new reset count for free call user command object
func newFreeCallResetCountCommand(cmd *cobra.Command, args []string, pComponents *Components) (command Command, err error) {
	userID, address, err := getFreeCallIDs(cmd)
	if err != nil {
		return
	}
	command = &freeCallUserResetCountCommand{
		userStorage: pComponents.FreeCallUserStorage(),
		userID:      userID,
		address:     address,
		orgMetadata: pComponents.OrganizationMetaData(),
	}
	return
}

func getFreeCallIDs(*cobra.Command) (userId, address string, err error) {
	return freeCallUserId, freeCallAddress, nil
}

// command's run method
func (command *freeCallUserUnLockCommand) Run() (err error) {
	if command.userId == "" {
		return fmt.Errorf("unlock -u or unlock -user-id must be set")
	}
	return command.unlockFreeCallUser()
}

// command's run method
func (command *freeCallUserResetCountCommand) Run() (err error) {
	if command.userID == "" {
		return fmt.Errorf("reset -u or reset -user-id must be set")
	}
	return command.resetUserForFreeCalls()
}

// release the lock on the user with the given user id
func (command *freeCallUserUnLockCommand) unlockFreeCallUser() (err error) {
	key := &escrow.FreeCallUserKey{}
	key.UserId = freeCallUserId
	key.OrganizationId = config.GetString(config.OrganizationId)
	key.ServiceId = config.GetString(config.ServiceId)
	key.GroupID = command.orgMetadata.GetGroupIdString()
	// check whether the key exists or not
	_, ok, err := command.lockStorage.Get(key.String())
	if !ok {
		fmt.Printf("Error: Free Call lock for user %s is not found\n", key.String())
		return
	}
	// try deleting the key
	err = command.lockStorage.Delete(key.String())
	if err != nil {
		fmt.Printf("Error: Unable to unlock the user -%s\n", key.String())
		return
	}
	fmt.Printf("Success: User %s unlocked\n", key.String())
	return
}

// reset free locks counter for a given user id
func (command *freeCallUserResetCountCommand) resetUserForFreeCalls() (err error) {
	key := &escrow.FreeCallUserKey{}
	key.UserId = freeCallUserId
	key.OrganizationId = config.GetString(config.OrganizationId)
	key.ServiceId = config.GetString(config.ServiceId)
	key.GroupID = command.orgMetadata.GetGroupIdString()
	// check whether the key exists or not
	_, ok, err := command.userStorage.Get(key)
	if !ok {
		fmt.Printf("Error: Free Call user %s is not found\n", key.String())
		return
	}
	updatedData := &escrow.FreeCallUserData{Address: key.Address, UserID: key.UserId, FreeCallsMade: 0}
	updatedData.OrganizationId = key.OrganizationId
	updatedData.ServiceId = key.ServiceId
	updatedData.GroupID = key.GroupID
	err = command.userStorage.Put(key, updatedData)
	if err != nil {
		fmt.Printf("Error: Unable to reset the user -%s\n", key.String())
		return
	}
	fmt.Printf("Success: User %s free calls have been reset \n", key.String())
	return
}

// ListFreeCallUserCmd displays all the users of free call for the given service and group
var ListFreeCallUserCmd = &cobra.Command{
	Use:   "list",
	Short: "List of all free call users",
	Long:  "List all users who have availed free calls to your service",
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
		fmt.Println("no users of free calls, yet in storage")
	}

	for _, user := range users {
		fmt.Printf("%v\n", user.String())
	}

	return nil
}
