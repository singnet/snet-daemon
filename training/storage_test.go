package training

import (
	"github.com/singnet/snet-daemon/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ModelStorageSuite struct {
	suite.Suite
	memoryStorage     *storage.MemoryStorage
	storage           *ModelStorage
	organizationId    string
	serviceId         string
	groupId           string
	methodName        string
	accessibleAddress []string
}

func (suite *ModelStorageSuite) getUserModelKey(modelId string) *ModelKey {
	return &ModelKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, ModelId: modelId, MethodName: suite.methodName}
}

func (suite *ModelStorageSuite) getUserModelData(modelId string) *ModelData {
	return &ModelData{
		Status:              "Created",
		ModelId:             modelId,
		OrganizationId:      suite.organizationId,
		ServiceId:           suite.serviceId,
		GroupId:             suite.groupId,
		MethodName:          suite.methodName,
		AuthorizedAddresses: suite.accessibleAddress,
		CreatedByAddress:    suite.accessibleAddress[0],
		UpdatedByAddress:    suite.accessibleAddress[1],
	}
}

func (suite *ModelStorageSuite) SetupSuite() {
	suite.memoryStorage = storage.NewMemStorage()
	suite.storage = NewModelStorage(suite.memoryStorage)
	suite.accessibleAddress = make([]string, 2)
	suite.accessibleAddress[0] = "ADD1"
	suite.accessibleAddress[1] = "ADD2"
}

func TestFreeCallUserStorageSuite(t *testing.T) {
	suite.Run(t, new(ModelStorageSuite))
}

func (suite *ModelStorageSuite) TestModelStorage_GetAll() {
	key1 := suite.getUserModelKey("1")
	key2 := suite.getUserModelKey("2")
	data1 := suite.getUserModelData("1")
	data2 := suite.getUserModelData("2")
	suite.storage.Put(key1, data1)
	suite.storage.Put(key2, data2)
	models, err := suite.storage.GetAll()
	assert.Equal(suite.T(), len(models), 2)
	assert.Equal(suite.T(), err, nil)
	assert.Equal(suite.T(), models[0].String(), suite.getUserModelData("1").String())
	assert.Equal(suite.T(), models[1].String(), suite.getUserModelData("2").String())
}

func (suite *ModelStorageSuite) TestModelStorage_PutIfAbsent() {
	key1 := suite.getUserModelKey("3")
	data1 := suite.getUserModelData("3")
	ok, err := suite.storage.PutIfAbsent(key1, data1)
	assert.Equal(suite.T(), ok, true)
	assert.Equal(suite.T(), err, nil)
}

func (suite *ModelStorageSuite) Test_serializeModelKey() {

}

func (suite *ModelStorageSuite) TestModelStorage_CompareAndSwap() {
	key1 := suite.getUserModelKey("1")
	data1 := suite.getUserModelData("1")
	data2 := suite.getUserModelData("2")
	suite.storage.Put(key1, data1)
	ok, err := suite.storage.CompareAndSwap(key1, data1, data2)
	assert.Equal(suite.T(), ok, true)
	assert.Equal(suite.T(), err, nil)
	data, ok, err := suite.storage.Get(key1)
	assert.Equal(suite.T(), ok, true)
	assert.Equal(suite.T(), err, nil)
	assert.Equal(suite.T(), data, data2)
}
