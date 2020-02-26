package escrow

import (
	"testing"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FreeCallServiceSuite struct {
	suite.Suite
	memoryStorage *memoryStorage
	storage       *FreeCallUserStorage
	service       FreeCallUserService
	metadata      *blockchain.ServiceMetadata
	groupId       [32]byte
}

func (suite *FreeCallServiceSuite) FreeCallUserData() *FreeCallUserData {
	return &FreeCallUserData{
		FreeCallsMade: 11,
	}
}
func (suite *FreeCallServiceSuite) SetupSuite() {
	metadata, err := blockchain.InitServiceMetaDataFromJson(testJsonData)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	suite.metadata = metadata
	suite.memoryStorage = NewMemStorage()
	suite.groupId = [32]byte{123}
	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
	suite.service = NewFreeCallUserService(suite.storage,
		NewEtcdLocker(suite.memoryStorage), func() ([32]byte, error) { return suite.groupId, nil },
		suite.metadata)
	userKey, err := suite.service.GetFreeCallUserKey(suite.payment("user1"))
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	err = suite.storage.Put(userKey, suite.FreeCallUserData())
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func TestFreeCallServiceSuite(t *testing.T) {
	suite.Run(t, new(FreeCallServiceSuite))
}

func (suite *FreeCallServiceSuite) payment(user string) *FreeCallPayment {
	payment := &FreeCallPayment{
		UserId:         user,
		ServiceId:      config.GetString(config.ServiceId),
		OrganizationId: config.GetString(config.OrganizationId),
	}

	return payment
}

func (suite *FreeCallServiceSuite) TestFreeCallUserTransaction() {
	payment := suite.payment("user1")
	userKey, err := suite.service.GetFreeCallUserKey(payment)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	freeCallUserDataBefore := suite.FreeCallUserData()
	IncrementFreeCallCount(freeCallUserDataBefore)
	transaction, errA := suite.service.StartFreeCallUserTransaction(payment)
	assert.Contains(suite.T(), transaction.(*freeCallTransaction).String(), "user1")
	errB := transaction.Commit()

	freeCallUserDataAfter, ok, errC := suite.storage.Get(userKey)

	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Nil(suite.T(), errB, "Unexpected error: %v", errB)
	assert.Nil(suite.T(), errC, "Unexpected error: %v", errC)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), freeCallUserDataAfter, freeCallUserDataBefore)
	transaction, errA = suite.service.StartFreeCallUserTransaction(payment)
	assert.NotNil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.Equal(suite.T(), "free call limit has been exceeded, calls made = 12,total free calls eligible = 12", errA.Error())
}

func (suite *FreeCallServiceSuite) TestFreeCallUserTransactionTestLock() {
	payment := suite.payment("user2")
	transactionA, errA := suite.service.StartFreeCallUserTransaction(payment)
	assert.Nil(suite.T(), errA, "Unexpected error: %v", errA)
	assert.NotNil(suite.T(), transactionA)
	transactionB, errB := suite.service.StartFreeCallUserTransaction(payment)
	assert.Nil(suite.T(), transactionB)
	assert.Equal(suite.T(), errB.Error(), "another transaction on this user: {ID:user2/ExampleOrganizationId/ExampleServiceId/ewAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=} is in progress")
}

func (suite *FreeCallServiceSuite) TestListFreeCallUsers() {
	users, err := suite.service.ListFreeCallUsers()
	assert.True(suite.T(), len(users) > 0)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}

func (suite *FreeCallServiceSuite) TestFreeCallUserTransactionRollBack() {
	payment := suite.payment("user4")
	userKey, err := suite.service.GetFreeCallUserKey(payment)
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
