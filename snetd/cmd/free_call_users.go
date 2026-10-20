package cmd

import (
	"fmt"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/freecall"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var FreeCallUserUnLockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock a free call user",
	Long: "allows us to perform operations on free call users with given address and optional user_id(email)." +
		" User can use 'snetd freecall unlock -a {address} -u {user-id}' command to unlock the User manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newFreeCallUserUnLockCommandCommand)
	},
}

var FreeCallUserResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the count on free calls used for the given user to zero",
	Long:  "User can use 'snetd freecall reset -a {address} -u {user-id}' command to reset the free call used of this User manually.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newFreeCallResetCountCommand)
	},
}

// freeCallUserUnLockCommand removes a user's shared-storage usage lock.
type freeCallUserUnLockCommand struct {
	lockStorage *storage.PrefixedAtomicStorage
	userID      string
	address     string
	orgMetadata *blockchain.OrganizationMetaData
}

// freeCallUserResetCountCommand resets a user's persisted free-call usage count.
type freeCallUserResetCountCommand struct {
	lockStorage *storage.PrefixedAtomicStorage
	userStorage *freecall.UserStorage
	userID      string
	address     string
	orgMetadata *blockchain.OrganizationMetaData
}

// newFreeCallUserUnLockCommandCommand selects the usage lock store and user identity.
func newFreeCallUserUnLockCommandCommand(cmd *cobra.Command, args []string, pComponents *Components) (command Command, err error) {
	userID, address, err := getFreeCallIDs(cmd)
	if err != nil {
		zap.L().Error(err.Error())
		return
	}
	command = &freeCallUserUnLockCommand{
		lockStorage: pComponents.FreeCallLockerStorage(),
		userID:      userID,
		address:     address,
		orgMetadata: pComponents.OrganizationMetaData(),
	}
	return
}

// newFreeCallResetCountCommand selects the usage record store and user identity.
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

func getFreeCallIDs(cmd *cobra.Command) (userId, address string, err error) {
	address, err = cmd.Flags().GetString(AddressFlag)
	if err != nil {
		return "", "", err
	}

	userId, err = cmd.Flags().GetString(UserIdFlag)
	if err != nil {
		return "", "", err
	}

	return userId, address, nil
}

// Run requires a wallet address; the user ID may be empty for wallet-only calls.
func (command *freeCallUserUnLockCommand) Run() (err error) {
	if command.address == "" {
		return fmt.Errorf("--address must be set (can be combined with --user-id)")
	}
	return command.unlockFreeCallUser()
}

// Run requires a wallet address; the user ID may be empty for wallet-only calls.
func (command *freeCallUserResetCountCommand) Run() (err error) {
	if command.address == "" {
		return fmt.Errorf("--address must be set (can be combined with --user-id)")
	}
	return command.resetUserForFreeCalls()
}

// unlockFreeCallUser deletes the lock for this address/user ID and configured
// organization/service/group. It does not reset usage or change any contract state.
func (command *freeCallUserUnLockCommand) unlockFreeCallUser() (err error) {
	key := &freecall.UserKey{}
	key.UserId = command.userID
	key.Address = command.address
	key.OrganizationId = config.GetString(config.OrganizationId)
	key.ServiceId = config.GetString(config.ServiceId)
	key.GroupID = command.orgMetadata.GetGroupIdString()
	// Look up the lock, not the user's usage record.
	_, ok, err := command.lockStorage.Get(key.String())
	if !ok {
		fmt.Printf("Error: Free Call lock for user %s is not found\n", key.String())
		return
	}
	// Delete only the lock key.
	err = command.lockStorage.Delete(key.String())
	if err != nil {
		fmt.Printf("Error: Unable to unlock the user -%s\n", key.String())
		return
	}
	fmt.Printf("Success: User %s unlocked\n", key.String())
	return
}

// resetUserForFreeCalls sets the persisted usage count to zero for this
// address/user ID and organization/service/group. It does not acquire or remove a lock.
func (command *freeCallUserResetCountCommand) resetUserForFreeCalls() (err error) {
	key := &freecall.UserKey{}
	key.UserId = command.userID
	key.Address = command.address
	key.OrganizationId = config.GetString(config.OrganizationId)
	key.ServiceId = config.GetString(config.ServiceId)
	key.GroupID = command.orgMetadata.GetGroupIdString()
	// check whether the key exists or not
	_, ok, err := command.userStorage.Get(key)
	if !ok {
		fmt.Printf("Error: Free Call user %s is not found\n", key.String())
		return
	}
	updatedData := &freecall.UserData{Address: key.Address, UserID: key.UserId, FreeCallsMade: 0}
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

// ListFreeCallUserCmd lists stored usage records in the selected network/organization/group
// namespace. Listing does not filter out records belonging to other services in that group.
var ListFreeCallUserCmd = &cobra.Command{
	Use:   "list",
	Short: "List of all free call users",
	Long:  "List all users who have availed free calls to your service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newListFreeCallUserCommand)
	},
}

type listFreeCallUsersCommand struct {
	freeCallService freecall.UserService
}

func newListFreeCallUserCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	command = &listFreeCallUsersCommand{
		freeCallService: components.FreeCallUserService(),
	}
	return
}

func (command *listFreeCallUsersCommand) Run() (err error) {
	users, err := command.freeCallService.ListUsers()
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
