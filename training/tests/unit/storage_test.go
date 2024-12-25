package tests

import (
	"fmt"
	"testing"

	base_storage "github.com/singnet/snet-daemon/v5/storage"
	"github.com/singnet/snet-daemon/v5/training"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelStorageSuite struct {
	suite.Suite
	memoryStorage     *base_storage.MemoryStorage
	storage           *training.ModelStorage
	userStorage       *training.ModelUserStorage
	pendingStorage    *training.PendingModelStorage
	organizationId    string
	serviceId         string
	groupId           string
	methodName        string
	accessibleAddress []string
}

func (suite *ModelStorageSuite) getModelKey(modelId string) *training.ModelKey {
	return &training.ModelKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, ModelId: modelId}
}

func (suite *ModelStorageSuite) getUserModelKey(address string) *training.ModelUserKey {
	return &training.ModelUserKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, UserAddress: address}
}

func (suite *ModelStorageSuite) getPendingModelKey() *training.PendingModelKey {
	return &training.PendingModelKey{OrganizationId: suite.organizationId, ServiceId: suite.serviceId,
		GroupId: suite.groupId}
}

func (suite *ModelStorageSuite) getModelData(modelId string) *training.ModelData {
	return &training.ModelData{
		Status:              training.Status_CREATED,
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

func (suite *ModelStorageSuite) getUserModelData(modelId []string) *training.ModelUserData {
	return &training.ModelUserData{
		ModelIds:       modelId,
		OrganizationId: suite.organizationId,
		ServiceId:      suite.serviceId,
		GroupId:        suite.groupId,
		// GRPCMethodName: suite.methodName,
	}
}

func (suite *ModelStorageSuite) getPendingModelData(modelIds []string) *training.PendingModelData {
	return &training.PendingModelData{
		ModelIDs: modelIds,
	}
}

func (suite *ModelStorageSuite) SetupSuite() {
	suite.memoryStorage = base_storage.NewMemStorage()
	suite.storage = training.NewModelStorage(suite.memoryStorage)
	suite.userStorage = training.NewUserModelStorage(base_storage.NewMemStorage())
	suite.pendingStorage = training.NewPendingModelStorage(base_storage.NewMemStorage())
	suite.accessibleAddress = make([]string, 2)
	suite.accessibleAddress[0] = "ADD1"
	suite.accessibleAddress[1] = "ADD2"
	suite.organizationId = "org_id"
	suite.serviceId = "service_id"
	suite.groupId = "group_id"
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
	modelId := "1"
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId, modelId)

	key := suite.getModelKey(modelId)
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
}

func (suite *ModelStorageSuite) Test_serializeUserModelKey() {
	userAddress := "test_address"
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId, userAddress)

	key := suite.getUserModelKey(userAddress)
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
}

func (suite *ModelStorageSuite) Test_serializePendingModelKey() {
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId)

	key := suite.getPendingModelKey()
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
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
	suite.userStorage.Put(key1, data1)
	suite.userStorage.Put(key2, data2)
	models, err := suite.userStorage.GetAll()
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
	suite.userStorage.PutIfAbsent(key1, data3)
	retrieveddata, ok, err := suite.userStorage.Get(key1)
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), retrieveddata, suite.getUserModelData([]string{"1"}))

	suite.userStorage.PutIfAbsent(key3, data3)
	retrieveddata, ok, err = suite.userStorage.Get(key3)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), retrieveddata, suite.getUserModelData([]string{"3"}))

	ok, err = suite.userStorage.CompareAndSwap(key1, data1, data3)
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), err)
	retrieveddata, ok, err = suite.userStorage.Get(key1)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), retrieveddata, suite.getUserModelData([]string{"3"}))

	_, ok, err = suite.userStorage.Get(suite.getUserModelKey("4"))
	assert.Equal(suite.T(), ok, false)
}

func (suite *ModelStorageSuite) TestPendingModelStorage_Get() {
	key := suite.getPendingModelKey()
	data := suite.getPendingModelData([]string{"1", "2"})

	err := suite.pendingStorage.Put(key, data)
	assert.NoError(suite.T(), err)

	newData, ok, err := suite.pendingStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), data, newData)
}
