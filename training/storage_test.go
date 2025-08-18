package training

import (
	"fmt"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	basestorage "github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelStorageSuite struct {
	suite.Suite
	memoryStorage        *basestorage.MemoryStorage
	storage              *ModelStorage
	userStorage          *ModelUserStorage
	pendingStorage       *PendingModelStorage
	publicStorage        *PublicModelStorage
	organizationMetaData *blockchain.OrganizationMetaData
	organizationId       string
	serviceId            string
	groupId              string
	methodName           string
	accessibleAddress    []string
}

func (suite *ModelStorageSuite) getModelKey(modelId string) *ModelKey {
	return &ModelKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, ModelId: modelId}
}

func (suite *ModelStorageSuite) getUserModelKey(address string) *ModelUserKey {
	return &ModelUserKey{OrganizationId: suite.organizationId, GroupId: suite.groupId,
		ServiceId: suite.serviceId, UserAddress: address}
}

func (suite *ModelStorageSuite) getPendingModelKey() *PendingModelKey {
	return &PendingModelKey{OrganizationId: suite.organizationId, ServiceId: suite.serviceId,
		GroupId: suite.groupId}
}

func (suite *ModelStorageSuite) getPublicModelKey() *PublicModelKey {
	return &PublicModelKey{OrganizationId: suite.organizationId, ServiceId: suite.serviceId,
		GroupId: suite.groupId}
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
	}
}

func (suite *ModelStorageSuite) getPendingModelData(modelIds []string) *PendingModelData {
	return &PendingModelData{
		ModelIDs: modelIds,
	}
}

func (suite *ModelStorageSuite) getPublicModelData(modelIds []string) *PublicModelData {
	return &PublicModelData{
		ModelIDs: modelIds,
	}
}

var testJsonOrgMeta = "{\n    \"org_name\": \"semyon_dev\",\n    \"org_id\": \"semyon_dev\",\n    \"org_type\": \"individual\",\n    \"description\": {\n        \"description\": \"Describe your organization details here\",\n        \"short_description\": \"This is short description of your organization\",\n        \"url\": \"https://anyurlofyourorganization\"\n    },\n    \"assets\": {},\n    \"contacts\": [],\n    \"groups\": [\n        {\n            \"group_name\": \"default_group\",\n            \"group_id\": \"FtNuizEOUsVCd5f2Fij9soehtRSb58LlTePgkVnsgVI=\",\n            \"payment\": {\n                \"payment_address\": \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\",\n                \"payment_expiration_threshold\": 40320,\n                \"payment_channel_storage_type\": \"etcd\",\n                \"payment_channel_storage_client\": {\n                    \"connection_timeout\": \"10s\",\n                    \"request_timeout\": \"5s\",\n                    \"endpoints\": [\n                        \"http://0.0.0.0:2379\"\n                    ]\n                }\n            }\n        },\n        {\n            \"group_name\": \"not_default\",\n            \"group_id\": \"udN0SLIvsDdvQQe3Ltv/NwqCh7sPKdz4scYmlI7AMdE=\",\n            \"payment\": {\n                \"payment_address\": \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\",\n                \"payment_expiration_threshold\": 100,\n                \"payment_channel_storage_type\": \"etcd\",\n                \"payment_channel_storage_client\": {\n                    \"connection_timeout\": \"7s\",\n                    \"request_timeout\": \"5s\",\n                    \"endpoints\": [\n                        \"http://0.0.0.0:2379\"\n                    ]\n                }\n            }\n        }\n    ]\n}"

func (suite *ModelStorageSuite) SetupSuite() {
	metadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgMeta))
	if err != nil {
		panic(err)
	}
	suite.memoryStorage = basestorage.NewMemStorage()
	suite.organizationMetaData = metadata
	suite.storage = NewModelStorage(suite.memoryStorage, suite.organizationMetaData)
	suite.userStorage = NewUserModelStorage(suite.memoryStorage, suite.organizationMetaData)
	suite.pendingStorage = NewPendingModelStorage(suite.memoryStorage, suite.organizationMetaData)
	suite.publicStorage = NewPublicModelStorage(suite.memoryStorage, suite.organizationMetaData)
	suite.accessibleAddress = make([]string, 2)
	suite.accessibleAddress[0] = "ADD1"
	suite.accessibleAddress[1] = "ADD2"
	suite.organizationId = "semyon_dev"
	suite.serviceId = "semyon_dev"
	suite.groupId = "FtNuizEOUsVCd5f2Fij9soehtRSb58LlTePgkVnsgVI="
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

func (suite *ModelStorageSuite) TestSerializeModelKey() {
	modelId := "1"
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId, modelId)

	key := suite.getModelKey(modelId)
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
}

func (suite *ModelStorageSuite) TestSerializeUserModelKey() {
	userAddress := "test_address"
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId, userAddress)

	key := suite.getUserModelKey(userAddress)
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
}

func (suite *ModelStorageSuite) TestSerializePendingModelKey() {
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId)

	key := suite.getPendingModelKey()
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
}

func (suite *ModelStorageSuite) TestSerializePublicModelKey() {
	expectedSerializedKey := fmt.Sprintf("{ID:%v|%v|%v}", suite.organizationId, suite.serviceId, suite.groupId)

	key := suite.getPublicModelKey()
	serializedKey := key.String()

	assert.Equal(suite.T(), expectedSerializedKey, serializedKey)
}

func (suite *ModelStorageSuite) TestModelStorage_CompareAndSwap() {
	key1 := suite.getModelKey("1")
	data1 := suite.getModelData("1")
	data2 := suite.getModelData("2")
	err := suite.storage.Put(key1, data1)
	assert.Nil(suite.T(), err)
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

func (suite *ModelStorageSuite) TestPendingModelStorage_AddPendingModelId() {
	key := suite.getPendingModelKey()
	data := suite.getPendingModelData([]string{"1", "2"})

	err := suite.pendingStorage.Put(key, data)
	assert.NoError(suite.T(), err)

	newData, ok, err := suite.pendingStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), data, newData)

	newModelId := "3"
	data = suite.getPendingModelData([]string{"1", "2", "3"})
	err = suite.pendingStorage.AddPendingModelId(key, newModelId)
	assert.NoError(suite.T(), err)
	newData, ok, err = suite.pendingStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), data, newData)
}

func (suite *ModelStorageSuite) TestPendingModelStorage_AddRemovePendingModelId() {
	key := suite.getPendingModelKey()
	data := suite.getPendingModelData([]string{"1", "2"})

	err := suite.pendingStorage.Put(key, data)
	assert.NoError(suite.T(), err)

	newData, ok, err := suite.pendingStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), data, newData)

	newModelId := "3"
	data = suite.getPendingModelData([]string{"1", "2", "3"})
	err = suite.pendingStorage.AddPendingModelId(key, newModelId)
	assert.NoError(suite.T(), err)
	newData, ok, err = suite.pendingStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), data, newData)

	data = suite.getPendingModelData([]string{"2", "3"})
	err = suite.pendingStorage.RemovePendingModelId(key, "1")
	assert.NoError(suite.T(), err)
	newData, ok, err = suite.pendingStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), data, newData)
}

func (suite *ModelStorageSuite) TestPublicModelStorage_Get() {
	key := suite.getPublicModelKey()
	data := suite.getPublicModelData([]string{"1", "2"})

	err := suite.publicStorage.Put(key, data)
	assert.NoError(suite.T(), err)

	newData, ok, err := suite.publicStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), data, newData)
}

func (suite *ModelStorageSuite) TestPublicModelStorage_AddPublicModelId() {
	key := suite.getPublicModelKey()
	data := suite.getPublicModelData([]string{"1", "2"})

	err := suite.publicStorage.Put(key, data)
	assert.NoError(suite.T(), err)

	newData, ok, err := suite.publicStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), data, newData)

	newModelId := "3"
	data = suite.getPublicModelData([]string{"1", "2", "3"})
	err = suite.publicStorage.AddPublicModelId(key, newModelId)
	assert.NoError(suite.T(), err)
	newData, ok, err = suite.publicStorage.Get(key)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), data, newData)
}
