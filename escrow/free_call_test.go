package escrow

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/storage"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

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
