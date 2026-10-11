package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const freeCallOrgGroupJSON = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"

func setFreeCallTestConfig(t *testing.T) {
	t.Helper()
	config.Vip().Set(config.OrganizationId, "test_org")
	config.Vip().Set(config.ServiceId, "test_service")
	config.Vip().Set(config.DaemonGroupName, "default_group")
}

func freeCallOrgMetadata(t *testing.T) *blockchain.OrganizationMetaData {
	t.Helper()
	metadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(freeCallOrgGroupJSON))
	require.NoError(t, err)
	require.NotNil(t, metadata)
	return metadata
}

func freeCallTestCmd(userID, address string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().StringP(AddressFlag, "a", "", "")
	cmd.Flags().StringP(UserIdFlag, "u", "", "")
	if address != "" {
		cmd.Flags().Set(AddressFlag, address)
	}
	if userID != "" {
		cmd.Flags().Set(UserIdFlag, userID)
	}
	return cmd
}

// failingAtomicStorage always returns the configured error
type failingAtomicStorage struct {
	err error
}

func (s *failingAtomicStorage) Get(key string) (value string, ok bool, err error) {
	return "", false, s.err
}
func (s *failingAtomicStorage) GetByKeyPrefix(prefix string) ([]string, error) {
	return nil, s.err
}
func (s *failingAtomicStorage) Put(key, value string) error { return s.err }
func (s *failingAtomicStorage) PutIfAbsent(key, value string) (bool, error) {
	return false, s.err
}
func (s *failingAtomicStorage) CompareAndSwap(key, prevValue, newValue string) (bool, error) {
	return false, s.err
}
func (s *failingAtomicStorage) Delete(key string) error { return s.err }
func (s *failingAtomicStorage) StartTransaction(conditionKeys []string) (storage.Transaction, error) {
	return nil, s.err
}
func (s *failingAtomicStorage) CompleteTransaction(transaction storage.Transaction, update []storage.KeyValueData) (bool, error) {
	return false, s.err
}
func (s *failingAtomicStorage) ExecuteTransaction(request storage.CASRequest) (bool, error) {
	return false, s.err
}

// stubFreeCallUserService is a mock of escrow.FreeCallUserService
type stubFreeCallUserService struct {
	users    []*escrow.FreeCallUserData
	usersErr error
}

func (s *stubFreeCallUserService) GetFreeCallUserKey(payment *escrow.FreeCallPayment) (*escrow.FreeCallUserKey, error) {
	return nil, nil
}
func (s *stubFreeCallUserService) FreeCallUser(key *escrow.FreeCallUserKey) (*escrow.FreeCallUserData, bool, error) {
	return nil, false, nil
}
func (s *stubFreeCallUserService) ListFreeCallUsers() ([]*escrow.FreeCallUserData, error) {
	return s.users, s.usersErr
}
func (s *stubFreeCallUserService) StartFreeCallUserTransaction(payment *escrow.FreeCallPayment) (escrow.FreeCallTransaction, error) {
	return nil, nil
}

func TestNewFreeCallUserUnLockCommandCommand(t *testing.T) {
	setFreeCallTestConfig(t)

	t.Run("flags not defined returns error", func(t *testing.T) {
		command, err := newFreeCallUserUnLockCommandCommand(&cobra.Command{}, nil, &Components{})
		assert.Error(t, err)
		assert.Nil(t, command)
	})

	t.Run("flags set builds command", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		components := &Components{
			freeCallLockerStorage: storage.NewPrefixedAtomicStorage(memStorage, "/free-call-locker"),
			organizationMetaData:  freeCallOrgMetadata(t),
		}

		command, err := newFreeCallUserUnLockCommandCommand(
			freeCallTestCmd("user@example.com", "0x1234"), nil, components)
		require.NoError(t, err)
		require.NotNil(t, command)

		unlockCommand, ok := command.(*freeCallUserUnLockCommand)
		require.True(t, ok)
		assert.Equal(t, "user@example.com", unlockCommand.userID)
		assert.Equal(t, "0x1234", unlockCommand.address)
		assert.NotNil(t, unlockCommand.lockStorage)
		assert.NotNil(t, unlockCommand.orgMetadata)
	})
}

func TestNewFreeCallResetCountCommand(t *testing.T) {
	setFreeCallTestConfig(t)

	t.Run("flags set builds command", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		components := &Components{
			freeCallUserStorage:  escrow.NewFreeCallUserStorage(memStorage),
			organizationMetaData: freeCallOrgMetadata(t),
		}

		command, err := newFreeCallResetCountCommand(
			freeCallTestCmd("user@example.com", "0x1234"), nil, components)
		require.NoError(t, err)
		require.NotNil(t, command)

		resetCommand, ok := command.(*freeCallUserResetCountCommand)
		require.True(t, ok)
		assert.Equal(t, "user@example.com", resetCommand.userID)
		assert.Equal(t, "0x1234", resetCommand.address)
		assert.NotNil(t, resetCommand.userStorage)
	})
}

func TestFreeCallUserUnLockCommandRunMissingAddress(t *testing.T) {
	command := &freeCallUserUnLockCommand{}
	err := command.Run()
	assert.EqualError(t, err, "--address must be set (can be combined with --user-id)")
}

func TestFreeCallUserResetCountCommandRunMissingAddress(t *testing.T) {
	command := &freeCallUserResetCountCommand{}
	err := command.Run()
	assert.EqualError(t, err, "--address must be set (can be combined with --user-id)")
}

func freeCallUserKey(address, userID string) *escrow.FreeCallUserKey {
	return &escrow.FreeCallUserKey{
		UserId:         userID,
		Address:        address,
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        freeCallOrgGroupID,
	}
}

const freeCallOrgGroupID = "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U="

func TestUnlockFreeCallUser(t *testing.T) {
	setFreeCallTestConfig(t)

	t.Run("unlock success removes existing lock", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		lockStorage := storage.NewPrefixedAtomicStorage(memStorage, "/free-call-locker")
		key := freeCallUserKey("0x1234", "user@example.com")
		require.NoError(t, lockStorage.Put(key.String(), "locked"))

		command := &freeCallUserUnLockCommand{
			lockStorage: lockStorage,
			address:     key.Address,
			userID:      key.UserId,
			orgMetadata: freeCallOrgMetadata(t),
		}

		err := command.Run()
		assert.NoError(t, err)

		_, ok, err := command.lockStorage.Get(key.String())
		require.NoError(t, err)
		assert.False(t, ok, "lock should be removed after unlock")
	})

	t.Run("unlock when lock is not found returns nil", func(t *testing.T) {
		command := &freeCallUserUnLockCommand{
			lockStorage: storage.NewPrefixedAtomicStorage(storage.NewMemStorage(), "/free-call-locker"),
			address:     "0x1234",
			userID:      "user@example.com",
			orgMetadata: freeCallOrgMetadata(t),
		}

		err := command.Run()
		assert.NoError(t, err, "unlock of a non-locked user should not return an error")
	})

	t.Run("unlock storage error is returned", func(t *testing.T) {
		storageErr := errors.New("storage down")
		command := &freeCallUserUnLockCommand{
			lockStorage: storage.NewPrefixedAtomicStorage(&failingAtomicStorage{err: storageErr}, "/free-call-locker"),
			address:     "0x1234",
			userID:      "user@example.com",
			orgMetadata: freeCallOrgMetadata(t),
		}

		err := command.Run()
		assert.Error(t, err)
		assert.Equal(t, storageErr, err)
	})
}

func TestResetUserForFreeCalls(t *testing.T) {
	setFreeCallTestConfig(t)

	t.Run("reset success sets count to zero", func(t *testing.T) {
		memStorage := storage.NewMemStorage()
		userStorage := escrow.NewFreeCallUserStorage(memStorage)
		key := freeCallUserKey("0x1234", "user@example.com")
		require.NoError(t, userStorage.Put(key, &escrow.FreeCallUserData{
			Address:        key.Address,
			UserID:         key.UserId,
			OrganizationId: key.OrganizationId,
			ServiceId:      key.ServiceId,
			GroupID:        key.GroupID,
			FreeCallsMade:  5,
		}))

		command := &freeCallUserResetCountCommand{
			userStorage: userStorage,
			address:     key.Address,
			userID:      key.UserId,
			orgMetadata: freeCallOrgMetadata(t),
		}

		err := command.Run()
		assert.NoError(t, err)

		user, ok, err := userStorage.Get(key)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, 0, user.FreeCallsMade)
	})

	t.Run("reset when user is not found returns nil", func(t *testing.T) {
		command := &freeCallUserResetCountCommand{
			userStorage: escrow.NewFreeCallUserStorage(storage.NewMemStorage()),
			address:     "0x1234",
			userID:      "user@example.com",
			orgMetadata: freeCallOrgMetadata(t),
		}

		err := command.Run()
		assert.NoError(t, err, "reset of unknown user should not return an error")
	})

	t.Run("reset storage error is returned", func(t *testing.T) {
		storageErr := errors.New("storage down")
		command := &freeCallUserResetCountCommand{
			userStorage: escrow.NewFreeCallUserStorage(&failingAtomicStorage{err: storageErr}),
			address:     "0x1234",
			userID:      "user@example.com",
			orgMetadata: freeCallOrgMetadata(t),
		}

		err := command.Run()
		assert.Error(t, err)
		assert.Equal(t, storageErr, err)
	})
}

func TestNewListFreeCallUserCommand(t *testing.T) {
	freeCallService := &stubFreeCallUserService{}
	components := &Components{freeCallUserService: freeCallService}

	command, err := newListFreeCallUserCommand(&cobra.Command{}, nil, components)
	require.NoError(t, err)
	require.NotNil(t, command)

	listCommand, ok := command.(*listFreeCallUsersCommand)
	require.True(t, ok)
	assert.Equal(t, freeCallService, listCommand.freeCallService)
}

func TestListFreeCallUsersCommandRun(t *testing.T) {
	t.Run("no users prints empty message", func(t *testing.T) {
		command := &listFreeCallUsersCommand{
			freeCallService: &stubFreeCallUserService{},
		}
		assert.NoError(t, command.Run())
	})

	t.Run("lists all users", func(t *testing.T) {
		users := []*escrow.FreeCallUserData{
			{Address: "0x1234", UserID: "user1@example.com", FreeCallsMade: 1},
			{Address: "0x5678", UserID: "user2@example.com", FreeCallsMade: 2},
		}
		command := &listFreeCallUsersCommand{
			freeCallService: &stubFreeCallUserService{users: users},
		}
		assert.NoError(t, command.Run())
	})

	t.Run("service error is returned", func(t *testing.T) {
		expected := fmt.Errorf("list error")
		command := &listFreeCallUsersCommand{
			freeCallService: &stubFreeCallUserService{usersErr: expected},
		}
		err := command.Run()
		assert.Equal(t, expected, err)
	})
}
