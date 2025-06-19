package escrow

import (
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type FreeCallUserStorageSuite struct {
	suite.Suite
	memoryStorage *storage.MemoryStorage
	storage       *FreeCallUserStorage
}

func (suite *FreeCallUserStorageSuite) SetupSuite() {

	suite.memoryStorage = storage.NewMemStorage()

	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
}

func (suite *FreeCallUserStorageSuite) SetupTest() {
	suite.memoryStorage.Clear()
}

func (suite *FreeCallUserStorageSuite) getFreeCallUserKey(userid string) *FreeCallUserKey {
	return &FreeCallUserKey{UserId: userid, GroupID: "Group1",
		ServiceId: "service1", OrganizationId: "org1"}
}

func (suite *FreeCallUserStorageSuite) getFreeCallUser() *FreeCallUserData {
	return &FreeCallUserData{
		FreeCallsMade: 1,
	}
}
func TestFreeCallUserStorageSuite(t *testing.T) {
	suite.Run(t, new(FreeCallUserStorageSuite))
}

func (suite *FreeCallUserStorageSuite) TestGetAll() {

	users, err := suite.storage.GetAll()
	assert.Equal(suite.T(), len(users), 0)
	userA := suite.getFreeCallUser()
	err = suite.storage.Put(suite.getFreeCallUserKey("userA"), userA)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	userB := suite.getFreeCallUser()
	err = suite.storage.Put(suite.getFreeCallUserKey("userB"), userB)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	users, err = suite.storage.GetAll()

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.Equal(suite.T(), []*FreeCallUserData{userA, userB}, users)
}

func (suite *FreeCallUserStorageSuite) TestGet() {
	userA, ok, err := suite.storage.Get(suite.getFreeCallUserKey("userA"))

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.False(suite.T(), ok)
	assert.Nil(suite.T(), userA)
}

func (suite *FreeCallUserStorageSuite) TestPutIfAbsent() {
	ok, err := suite.storage.PutIfAbsent(suite.getFreeCallUserKey("userC"), suite.getFreeCallUser())

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.True(suite.T(), ok)

	userC, ok, err := suite.storage.Get(suite.getFreeCallUserKey("userC"))
	assert.Equal(suite.T(), suite.getFreeCallUser(), userC)
}

func (suite *FreeCallUserStorageSuite) TestCompareAndSwap() {
	ok, err := suite.storage.PutIfAbsent(suite.getFreeCallUserKey("userD"), suite.getFreeCallUser())
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.True(suite.T(), ok)
	freeCallUser2 := suite.getFreeCallUser()
	IncrementFreeCallCount(freeCallUser2)
	ok, err = suite.storage.CompareAndSwap(suite.getFreeCallUserKey("userD"), suite.getFreeCallUser(), freeCallUser2)

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.True(suite.T(), ok)

	userC, ok, err := suite.storage.Get(suite.getFreeCallUserKey("userD"))
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), 2, userC.FreeCallsMade)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
}
