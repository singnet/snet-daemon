package escrow

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	coverageTestAddress  = "0x101a018fe784bf01d538b5d4dfa311d047dba491"
	coverageTestGroupID  = "ewAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	coverageTestUserID   = "user-1"
	defaultCoverageFails = 10
)

var coverageTestGroupIDBytes = [32]byte{123}

var coverageMetadataJson = `{
	"version": 1,
	"display_name": "Example1",
	"encoding": "grpc",
	"service_type": "grpc",
	"payment_expiration_threshold": 40320,
	"mpe_address": "0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08",
	"groups": [
		{
			"free_calls": 10,
			"free_call_signer_address": "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F",
			"endpoints": ["http://34.344.33.1:2379"],
			"group_id": "88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
			"group_name": "default_group",
			"pricing": [
				{"price_model": "fixed_price", "default": true, "price_in_cogs": 2}
			]
		}
	]
}`

func coverageMetadata(t *testing.T) *blockchain.ServiceMetadata {
	t.Helper()
	config.Vip().Set(config.DaemonGroupName, "default_group")
	metadata, err := blockchain.InitServiceMetaDataFromJson([]byte(coverageMetadataJson))
	require.NoError(t, err)
	require.Equal(t, defaultCoverageFails, metadata.GetFreeCallsAllowed())
	return metadata
}

func coverageGroupIDReader() ([32]byte, error) {
	return coverageTestGroupIDBytes, nil
}

func coveragePayment() *FreeCallPayment {
	return &FreeCallPayment{
		Address:        coverageTestAddress,
		UserID:         coverageTestUserID,
		ServiceId:      config.GetString(config.ServiceId),
		OrganizationId: config.GetString(config.OrganizationId),
		GroupId:        coverageTestGroupID,
	}
}

func coverageUserData(freeCallsMade int) *FreeCallUserData {
	return &FreeCallUserData{
		Address:        coverageTestAddress,
		UserID:         coverageTestUserID,
		FreeCallsMade:  freeCallsMade,
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        coverageTestGroupID,
	}
}

func coverageService(t *testing.T, s storage.AtomicStorage, locker Locker) FreeCallUserService {
	t.Helper()
	return NewFreeCallUserService(NewFreeCallUserStorage(s), locker, coverageGroupIDReader, coverageMetadata(t))
}

func putCoverageUser(t *testing.T, service FreeCallUserService, userStorage *FreeCallUserStorage, made int) *FreeCallUserKey {
	t.Helper()
	userKey, err := service.GetFreeCallUserKey(coveragePayment())
	require.NoError(t, err)
	require.NoError(t, userStorage.Put(userKey, coverageUserData(made)))
	return userKey
}

func restoreFreeCallsPerAddress(t *testing.T) {
	t.Helper()
	previous := config.Vip().Get(config.FreeCallsPerAddress)
	t.Cleanup(func() {
		config.Vip().Set(config.FreeCallsPerAddress, previous)
	})
}

// failingStorage is an AtomicStorage that can be forced to fail on Get, Put or
// CompareAndSwap operations. All other methods are delegated to MemoryStorage.
type failingStorage struct {
	*storage.MemoryStorage
	getErr error
	putErr error
	casErr error
}

func (f *failingStorage) Get(key string) (value string, ok bool, err error) {
	if f.getErr != nil {
		return "", false, f.getErr
	}
	return f.MemoryStorage.Get(key)
}

func (f *failingStorage) GetByKeyPrefix(prefix string) (values []string, err error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.MemoryStorage.GetByKeyPrefix(prefix)
}

func (f *failingStorage) Put(key, value string) (err error) {
	if f.putErr != nil {
		return f.putErr
	}
	return f.MemoryStorage.Put(key, value)
}

func (f *failingStorage) PutIfAbsent(key, value string) (ok bool, err error) {
	if f.putErr != nil {
		return false, f.putErr
	}
	return f.MemoryStorage.PutIfAbsent(key, value)
}

func (f *failingStorage) CompareAndSwap(key, prevValue, newValue string) (ok bool, err error) {
	if f.casErr != nil {
		return false, f.casErr
	}
	return f.MemoryStorage.CompareAndSwap(key, prevValue, newValue)
}

type mockLock struct {
	unlockErr error
}

func (l mockLock) Unlock() error {
	return l.unlockErr
}

type mockLocker struct {
	lock Lock
	ok   bool
	err  error
}

func (l mockLocker) Lock(name string) (lock Lock, ok bool, err error) {
	return l.lock, l.ok, l.err
}

func TestFreeCallTransaction_GetSender(t *testing.T) {
	transaction := &freeCallTransaction{payment: *coveragePayment()}
	assert.Equal(t, common.HexToAddress(coverageTestAddress), transaction.GetSender())
}

func TestFreeCallTransaction_String(t *testing.T) {
	transaction := &freeCallTransaction{payment: *coveragePayment(), freeCallUser: coverageUserData(3)}
	str := transaction.String()
	assert.Contains(t, str, coveragePayment().String())
	assert.Contains(t, str, transaction.freeCallUser.String())
	assert.Contains(t, str, coverageTestAddress)
}

func TestFreeCallTransaction_FreeCallUser(t *testing.T) {
	data := coverageUserData(4)
	transaction := &freeCallTransaction{freeCallUser: data}
	assert.Same(t, data, transaction.FreeCallUser())
}

func TestFreeCallUser_DataNotFoundReturnsNewEntry(t *testing.T) {
	service := coverageService(t, storage.NewMemStorage(), NewEtcdLocker(storage.NewMemStorage()))

	key := &FreeCallUserKey{
		UserId:         "missing",
		Address:        "0x0000000000000000000000000000000000000001",
		OrganizationId: "org",
		ServiceId:      "svc",
		GroupID:        "grp",
	}

	data, ok, err := service.FreeCallUser(key)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 0, data.FreeCallsMade)
	assert.Equal(t, "missing", data.UserID)
	assert.Equal(t, key.Address, data.Address)
	assert.Equal(t, key.GroupID, data.GroupID)
	assert.Equal(t, key.ServiceId, data.ServiceId)
	assert.Equal(t, key.OrganizationId, data.OrganizationId)
}

func TestFreeCallUser_StorageError(t *testing.T) {
	faulty := &failingStorage{MemoryStorage: storage.NewMemStorage(), getErr: errors.New("storage get error")}
	service := coverageService(t, faulty, NewEtcdLocker(faulty))

	_, ok, err := service.FreeCallUser(&FreeCallUserKey{Address: coverageTestAddress})
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestGetFreeCallUserKey(t *testing.T) {
	service := coverageService(t, storage.NewMemStorage(), NewEtcdLocker(storage.NewMemStorage()))

	key, err := service.GetFreeCallUserKey(coveragePayment())
	require.NoError(t, err)
	assert.Equal(t, coverageTestAddress, key.Address)
	assert.Equal(t, coverageTestUserID, key.UserId)
	assert.Equal(t, config.GetString(config.ServiceId), key.ServiceId)
	assert.Equal(t, config.GetString(config.OrganizationId), key.OrganizationId)
	assert.Equal(t, coverageTestGroupID, key.GroupID)
}

func TestGetFreeCallUserKey_GroupIDReaderError(t *testing.T) {
	service := NewFreeCallUserService(NewFreeCallUserStorage(storage.NewMemStorage()),
		NewEtcdLocker(storage.NewMemStorage()),
		func() ([32]byte, error) { return [32]byte{}, errors.New("group id error") },
		coverageMetadata(t))

	_, err := service.GetFreeCallUserKey(coveragePayment())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group id error")
}

func TestListFreeCallUsers_Empty(t *testing.T) {
	service := coverageService(t, storage.NewMemStorage(), NewEtcdLocker(storage.NewMemStorage()))

	users, err := service.ListFreeCallUsers()
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestListFreeCallUsers_WithData(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	userKey, err := service.GetFreeCallUserKey(coveragePayment())
	require.NoError(t, err)
	require.NoError(t, userStorage.Put(userKey, coverageUserData(2)))

	users, err := service.ListFreeCallUsers()
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, 2, users[0].FreeCallsMade)
}

func TestStartFreeCallUserTransaction_GroupIDReaderError(t *testing.T) {
	service := NewFreeCallUserService(NewFreeCallUserStorage(storage.NewMemStorage()),
		NewEtcdLocker(storage.NewMemStorage()),
		func() ([32]byte, error) { return [32]byte{}, errors.New("group id error") },
		coverageMetadata(t))

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	assert.Nil(t, transaction)
	require.Error(t, err)
	paymentErr, ok := err.(*PaymentError)
	require.True(t, ok)
	assert.Equal(t, Internal, paymentErr.Code)
	assert.Contains(t, paymentErr.Message, "payment freeCallUserKey error")
}

func TestStartFreeCallUserTransaction_FreeCallUserStorageError(t *testing.T) {
	faulty := &failingStorage{MemoryStorage: storage.NewMemStorage(), getErr: errors.New("storage get error")}
	service := coverageService(t, faulty, NewEtcdLocker(faulty))

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	assert.Nil(t, transaction)
	require.Error(t, err)
	paymentErr, ok := err.(*PaymentError)
	require.True(t, ok)
	assert.Equal(t, Internal, paymentErr.Code)
	assert.Contains(t, paymentErr.Message, "payment freeCallUserData error")
}

func TestStartFreeCallUserTransaction_LockerError(t *testing.T) {
	service := NewFreeCallUserService(NewFreeCallUserStorage(storage.NewMemStorage()),
		mockLocker{err: errors.New("lock error")},
		coverageGroupIDReader,
		coverageMetadata(t))

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	assert.Nil(t, transaction)
	require.Error(t, err)
	paymentErr, ok := err.(*PaymentError)
	require.True(t, ok)
	assert.Equal(t, Internal, paymentErr.Code)
	assert.Contains(t, paymentErr.Message, "cannot get mutex for user")
}

func TestStartFreeCallUserTransaction_LockUnavailable(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	putCoverageUser(t, service, userStorage, 0)

	first, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, first)
	t.Cleanup(func() { _ = first.Rollback() })

	second, err := service.StartFreeCallUserTransaction(coveragePayment())
	assert.Nil(t, second)
	require.Error(t, err)
	paymentErr, ok := err.(*PaymentError)
	require.True(t, ok)
	assert.Equal(t, FailedPrecondition, paymentErr.Code)
	assert.Contains(t, paymentErr.Message, "another transaction on this user")
}

func TestStartFreeCallUserTransaction_LimitExceededFromMetadata(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	putCoverageUser(t, service, userStorage, defaultCoverageFails)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	assert.Nil(t, transaction)
	require.Error(t, err)
	assert.Equal(t, "free call limit has been exceeded, calls made = 10, total free calls eligible = 10", err.Error())
}

func TestStartFreeCallUserTransaction_LimitExceededFromPerAddressConfig(t *testing.T) {
	restoreFreeCallsPerAddress(t)
	config.Vip().Set(config.FreeCallsPerAddress, map[string]any{strings.ToLower(coverageTestAddress): 1})

	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	putCoverageUser(t, service, userStorage, 1)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	assert.Nil(t, transaction)
	require.Error(t, err)
	assert.Equal(t, "free call limit has been exceeded, calls made = 1, total free calls eligible = 1", err.Error())
}

func TestStartFreeCallUserTransaction_UnlimitedPerAddress(t *testing.T) {
	restoreFreeCallsPerAddress(t)
	config.Vip().Set(config.FreeCallsPerAddress, map[string]any{strings.ToLower(coverageTestAddress): "unlimited"})

	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	putCoverageUser(t, service, userStorage, defaultCoverageFails+100)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)
	assert.Equal(t, coverageTestAddress, transaction.FreeCallUser().Address)

	require.NoError(t, transaction.Rollback())

	stored, ok, err := userStorage.Get(transaction.(*freeCallTransaction).freeCallUserKey)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, defaultCoverageFails+100, stored.FreeCallsMade)
}

func TestStartFreeCallUserTransaction_PopulatesServiceFieldsFromKey(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	userKey := putCoverageUser(t, service, userStorage, 0)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)
	t.Cleanup(func() { _ = transaction.Rollback() })

	user := transaction.FreeCallUser()
	assert.Equal(t, userKey.Address, user.Address)
	assert.Equal(t, userKey.ServiceId, user.ServiceId)
	assert.Equal(t, userKey.OrganizationId, user.OrganizationId)
	assert.Equal(t, userKey.GroupID, user.GroupID)
}

func TestFreeCallTransaction_Commit(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	userKey := putCoverageUser(t, service, userStorage, 3)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)

	require.NoError(t, transaction.Commit())

	stored, ok, err := userStorage.Get(userKey)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, 4, stored.FreeCallsMade)

	// Lock must be released after Commit.
	again, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, again)
	require.NoError(t, again.Rollback())
}

func TestFreeCallTransaction_CommitStorageError(t *testing.T) {
	faulty := &failingStorage{MemoryStorage: storage.NewMemStorage()}
	userStorage := NewFreeCallUserStorage(faulty)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(faulty), coverageGroupIDReader, coverageMetadata(t))

	putCoverageUser(t, service, userStorage, 1)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)

	faulty.putErr = errors.New("store failure")
	err = transaction.Commit()
	require.Error(t, err)
	paymentErr, ok := err.(*PaymentError)
	require.True(t, ok)
	assert.Equal(t, Internal, paymentErr.Code)
	assert.Contains(t, paymentErr.Message, "unable to store new transaction free call user state")
}

func TestFreeCallTransaction_CommitUnlockError(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage,
		mockLocker{lock: mockLock{unlockErr: errors.New("unlock failed")}, ok: true},
		coverageGroupIDReader,
		coverageMetadata(t))

	putCoverageUser(t, service, userStorage, 2)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)

	// Even if unlocking fails after the commit, the commit itself succeeds.
	assert.NoError(t, transaction.Commit())
}

func TestFreeCallTransaction_Rollback(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage, NewEtcdLocker(mem), coverageGroupIDReader, coverageMetadata(t))

	userKey := putCoverageUser(t, service, userStorage, 5)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)

	require.NoError(t, transaction.Rollback())

	stored, ok, err := userStorage.Get(userKey)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, 5, stored.FreeCallsMade)

	// Lock must be released after Rollback.
	again, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, again)
	require.NoError(t, again.Rollback())
}

func TestFreeCallTransaction_RollbackUnlockError(t *testing.T) {
	mem := storage.NewMemStorage()
	userStorage := NewFreeCallUserStorage(mem)
	service := NewFreeCallUserService(userStorage,
		mockLocker{lock: mockLock{unlockErr: errors.New("unlock failed")}, ok: true},
		coverageGroupIDReader,
		coverageMetadata(t))

	putCoverageUser(t, service, userStorage, 0)

	transaction, err := service.StartFreeCallUserTransaction(coveragePayment())
	require.NoError(t, err)
	require.NotNil(t, transaction)

	assert.NoError(t, transaction.Rollback())
}

type FreeCallServiceSuite struct {
	suite.Suite
	memoryStorage *storage.MemoryStorage
	storage       *FreeCallUserStorage
	userAddr      common.Address
	service       FreeCallUserService
	metadata      *blockchain.ServiceMetadata
	groupId       [32]byte
}

func (suite *FreeCallServiceSuite) FreeCallUserData(freeCallsMade int) *FreeCallUserData {
	return &FreeCallUserData{
		Address:        suite.userAddr.Hex(),
		UserID:         "",
		FreeCallsMade:  freeCallsMade,
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        "ewAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}
}

func (suite *FreeCallServiceSuite) SetupSuite() {
	metadata, err := blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	suite.metadata = metadata
	suite.memoryStorage = storage.NewMemStorage()
	suite.groupId = [32]byte{123}
	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
	suite.service = NewFreeCallUserService(suite.storage,
		NewEtcdLocker(suite.memoryStorage), func() ([32]byte, error) { return suite.groupId, nil },
		suite.metadata)

	ecdsa, err := crypto.HexToECDSA("aeaa9fb59c0dd868260af55ea65be077dbcaa063c067dfc0865845a0af5de84c")
	assert.Nil(suite.T(), err)
	suite.userAddr = crypto.PubkeyToAddress(ecdsa.PublicKey)
	assert.Nil(suite.T(), err)

	userKey, err := suite.service.GetFreeCallUserKey(suite.payment(suite.userAddr.Hex()))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	err = suite.storage.Put(userKey, suite.FreeCallUserData(8))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func TestFreeCallServiceSuite(t *testing.T) {
	suite.Run(t, new(FreeCallServiceSuite))
}

func (suite *FreeCallServiceSuite) payment(addr string) *FreeCallPayment {
	payment := &FreeCallPayment{
		Address:        addr,
		ServiceId:      config.GetString(config.ServiceId),
		OrganizationId: config.GetString(config.OrganizationId),
		GroupId:        "ewAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}
	return payment
}

func (suite *FreeCallServiceSuite) TestFreeCallUserTransaction() {
	payment := suite.payment(suite.userAddr.Hex())
	userKey, err := suite.service.GetFreeCallUserKey(payment)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)

	err = suite.storage.Put(userKey, suite.FreeCallUserData(9))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)

	freeCallUserDataBefore, _, err := suite.storage.Get(userKey)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	IncrementFreeCallCount(freeCallUserDataBefore) // 9+1=10

	transaction, errA := suite.service.StartFreeCallUserTransaction(payment)
	assert.Nil(suite.T(), errA)
	assert.Contains(suite.T(), transaction.(*freeCallTransaction).String(), suite.userAddr.Hex())
	errB := transaction.Commit()

	freeCallUserDataAfter, ok, errC := suite.storage.Get(userKey)

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	assert.Nil(suite.T(), errC, "Unexpected error: %v", errC)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), freeCallUserDataAfter, freeCallUserDataBefore)
	transaction, errA = suite.service.StartFreeCallUserTransaction(payment)
	assert.NotNil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Equal(suite.T(), "free call limit has been exceeded, calls made = 10, total free calls eligible = 10", errA.Error())
}

func (suite *FreeCallServiceSuite) TestFreeCallUserTransactionTestLock() {
	payment := suite.payment(suite.userAddr.Hex())
	userKey, err := suite.service.GetFreeCallUserKey(suite.payment(suite.userAddr.Hex()))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	err = suite.storage.Put(userKey, suite.FreeCallUserData(0))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)

	transactionA, errA := suite.service.StartFreeCallUserTransaction(payment)
	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.NotNil(suite.T(), transactionA)
	transactionB, errB := suite.service.StartFreeCallUserTransaction(payment)
	assert.Nil(suite.T(), transactionB)
	assert.NotNil(suite.T(), errB)
	assert.Equal(suite.T(), "another transaction on this user: {ID:0xF627CE8635cdC34b2f619FDDb4E4b61308D6BD68//YOUR_ORG_ID/YOUR_SERVICE_ID/ewAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=} is in progress", errB.Error())
}

func (suite *FreeCallServiceSuite) TestListFreeCallUsers() {
	users, err := suite.service.ListFreeCallUsers()
	assert.True(suite.T(), len(users) > 0)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *FreeCallServiceSuite) TestFreeCallUserTransactionRollBack() {
	payment := suite.payment(suite.userAddr.Hex())
	userKey, err := suite.service.GetFreeCallUserKey(payment)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)

	err = suite.storage.Put(userKey, suite.FreeCallUserData(0))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)

	userDataBefore, _, err := suite.service.FreeCallUser(userKey)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	transaction, errA := suite.service.StartFreeCallUserTransaction(payment)
	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.NotNil(suite.T(), transaction)
	errB := transaction.Rollback()
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	userDataAfter, _, err := suite.service.FreeCallUser(userKey)
	assert.Equal(suite.T(), userDataBefore.FreeCallsMade, userDataAfter.FreeCallsMade)
}
