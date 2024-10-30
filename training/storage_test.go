package training

import (
	"github.com/singnet/snet-daemon/v5/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ModelStorageSuite struct {
	suite.Suite
	memoryStorage     *storage.MemoryStorage
	storage           *ModelStorage
	userstorage       *ModelUserStorage
	organizationId    string
	serviceId         string
	groupId           string
	methodName        string
	accessibleAddress []string
}

func (suite *ModelStorageSuite) getModelKey(modelId string) *ModelKey {
	return &ModelKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, ModelId: modelId, GRPCMethodName: suite.methodName}
}
func (suite *ModelStorageSuite) getUserModelKey(address string) *ModelUserKey {
	return &ModelUserKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, GRPCMethodName: suite.methodName, UserAddress: address}
}

func (suite *ModelStorageSuite) getModelData(modelId string) *ModelData {
	return &ModelData{
		Status:              Status_CREATED,
		ModelId:             modelId,
		OrganizationId:      suite.organizationId,
		ServiceId:           suite.serviceId,
		GroupId:             suite.groupId,
		GRPCMethodName:      suite.methodName,
		AuthorizedAddresses: suite.accessibleAddress,
		CreatedByAddress:    suite.accessibleAddress[0],
		UpdatedByAddress:    suite.accessibleAddress[1],
	}
}

func (suite *ModelStorageSuite) getUserModelData(modelId []string) *ModelUserData {
	return &ModelUserData{
		ModelIds:       modelId,
		OrganizationId: suite.organizationId,
		ServiceId:      suite.serviceId,
		GroupId:        suite.groupId,
		GRPCMethodName: suite.methodName,
	}
}
func (suite *ModelStorageSuite) SetupSuite() {
	suite.memoryStorage = storage.NewMemStorage()
	suite.storage = NewModelStorage(suite.memoryStorage)
	suite.userstorage = NewUerModelStorage(storage.NewMemStorage())
	suite.accessibleAddress = make([]string, 2)
	suite.accessibleAddress[0] = "ADD1"
	suite.accessibleAddress[1] = "ADD2"
}

func TestFreeCallUserStorageSuite(t *testing.T) {
	suite.Run(t, new(ModelStorageSuite))
}

func (suite *ModelStorageSuite) TestModelStorage_GetAll() {

	key1 := suite.getModelKey("1")
	key2 := suite.getModelKey("2")
	data1 := suite.getModelData("1")
	data2 := suite.getModelData("2")
	suite.storage.Put(key1, data1)
	suite.storage.Put(key2, data2)
	models, err := suite.storage.GetAll()
	assert.Equal(suite.T(), len(models), 2)
	assert.Equal(suite.T(), err, nil)
	match1 := false
	match2 := false
	for _, model := range models {
		if model.String() == suite.getModelData("1").String() {
			match1 = true
		}
		if model.String() == suite.getModelData("2").String() {
			match2 = true
		}
	}
	assert.True(suite.T(), match2)
	assert.True(suite.T(), match1)
	_, ok, err := suite.storage.Get(suite.getModelKey("4"))
	assert.Equal(suite.T(), ok, false)

}

func (suite *ModelStorageSuite) TestModelStorage_PutIfAbsent() {
	key1 := suite.getModelKey("3")
	data1 := suite.getModelData("3")
	ok, err := suite.storage.PutIfAbsent(key1, data1)
	assert.Equal(suite.T(), ok, true)
	assert.Equal(suite.T(), err, nil)
}

func (suite *ModelStorageSuite) Test_serializeModelKey() {

}

func (suite *ModelStorageSuite) TestModelStorage_CompareAndSwap() {
	key1 := suite.getModelKey("1")
	data1 := suite.getModelData("1")
	data2 := suite.getModelData("2")
	suite.storage.Put(key1, data1)
	ok, err := suite.storage.CompareAndSwap(key1, data1, data2)
	assert.Equal(suite.T(), ok, true)
	assert.Equal(suite.T(), err, nil)
	data, ok, err := suite.storage.Get(key1)
	assert.Equal(suite.T(), ok, true)
	assert.Equal(suite.T(), err, nil)
	assert.Equal(suite.T(), data, data2)
}

func (suite *ModelStorageSuite) TestModelUserStorage_GetAll() {
	key1 := suite.getUserModelKey("1")
	key2 := suite.getUserModelKey("2")
	key3 := suite.getUserModelKey("3")
	data1 := suite.getUserModelData([]string{"1"})
	data2 := suite.getUserModelData([]string{"2"})
	data3 := suite.getUserModelData([]string{"3"})
	suite.userstorage.Put(key1, data1)
	suite.userstorage.Put(key2, data2)
	models, err := suite.userstorage.GetAll()
	assert.Equal(suite.T(), len(models), 2)
	assert.Equal(suite.T(), err, nil)
	match1 := false
	match2 := false
	for _, model := range models {
		if model.String() == suite.getUserModelData([]string{"1"}).String() {
			match1 = true
		}
		if model.String() == suite.getUserModelData([]string{"1"}).String() {
			match2 = true
		}
	}
	assert.True(suite.T(), match2)
	assert.True(suite.T(), match1)
	suite.userstorage.PutIfAbsent(key1, data3)
	retrieveddata, ok, err := suite.userstorage.Get(key1)
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), retrieveddata, suite.getUserModelData([]string{"1"}))

	suite.userstorage.PutIfAbsent(key3, data3)
	retrieveddata, ok, err = suite.userstorage.Get(key3)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), retrieveddata, suite.getUserModelData([]string{"3"}))

	ok, err = suite.userstorage.CompareAndSwap(key1, data1, data3)
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), err)
	retrieveddata, ok, err = suite.userstorage.Get(key1)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), retrieveddata, suite.getUserModelData([]string{"3"}))

	_, ok, err = suite.userstorage.Get(suite.getUserModelKey("4"))
	assert.Equal(suite.T(), ok, false)

}
