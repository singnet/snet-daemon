package freecall

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// FreeCallUserData is the persisted struct from escrow before issue #531.
// Keep this schema independent of UserData to detect storage incompatibilities.
type FreeCallUserData struct {
	Address        string
	UserID         string
	FreeCallsMade  int
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func TestLegacyUserStorageAndLocks(t *testing.T) {
	oldConfig := config.Vip()
	config.SetVip(viper.New())
	t.Cleanup(func() { config.SetVip(oldConfig) })

	legacy := FreeCallUserData{
		Address: coverageTestAddress, UserID: "user", FreeCallsMade: 9,
		OrganizationId: "org", ServiceId: "service", GroupID: coverageTestGroupID,
	}
	var encoded bytes.Buffer
	require.NoError(t, gob.NewEncoder(&encoded).Encode(&legacy))

	// Literal keys and lock values are those used by existing daemon instances.
	key := "{ID:" + coverageTestAddress + "/user/org/service/" + coverageTestGroupID + "}"
	dataKey := "/free-call-user/storage/" + key
	lockKey := "/freecall/lock/" + key
	mem := storage.NewMemStorage()
	require.NoError(t, mem.Put(dataKey, encoded.String()))
	require.NoError(t, mem.Put(lockKey, "locked"))

	users := NewUserStorage(mem)
	service := NewUserService(users, storage.NewLocker(storage.NewPrefixedAtomicStorage(mem, "/freecall/lock")),
		coverageGroupIDReader, coverageMetadata(t))
	payment := &Payment{Address: legacy.Address, UserID: legacy.UserID,
		OrganizationId: legacy.OrganizationId, ServiceId: legacy.ServiceId}

	// An old daemon's lock still prevents this instance from starting a call.
	transaction, err := service.StartTransaction(payment)
	require.Nil(t, transaction)
	var paymentErr *errs.PaymentError
	require.ErrorAs(t, err, &paymentErr)
	require.Equal(t, errs.FailedPrecondition, paymentErr.Code)

	require.NoError(t, mem.Put(lockKey, "unlocked"))
	transaction, err = service.StartTransaction(payment)
	require.NoError(t, err)
	require.Equal(t, 9, transaction.User().FreeCallsMade)
	require.NoError(t, transaction.Rollback())
	stored, ok, err := mem.Get(dataKey)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, encoded.String(), stored, "rollback must preserve existing usage")

	transaction, err = service.StartTransaction(payment)
	require.NoError(t, err)
	require.NoError(t, transaction.Commit())
	stored, ok, err = mem.Get(dataKey)
	require.NoError(t, err)
	require.True(t, ok)
	var updated FreeCallUserData
	require.NoError(t, gob.NewDecoder(bytes.NewBufferString(stored)).Decode(&updated))
	legacy.FreeCallsMade++
	require.Equal(t, legacy, updated, "old daemons must still be able to read updated usage")
	lock, ok, err := mem.Get(lockKey)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "unlocked", lock)

	transaction, err = service.StartTransaction(payment)
	require.Nil(t, transaction)
	require.EqualError(t, err, "free call limit has been exceeded, calls made = 10, total free calls eligible = 10")
}
