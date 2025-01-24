package training

import (
	"fmt"
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"go.uber.org/zap"
	"reflect"

	"github.com/singnet/snet-daemon/v5/storage"
	"github.com/singnet/snet-daemon/v5/utils"
)

type ModelStorage struct {
	delegate             storage.TypedAtomicStorage
	organizationMetaData *blockchain.OrganizationMetaData
}

type ModelUserStorage struct {
	delegate storage.TypedAtomicStorage
}

type PendingModelStorage struct {
	delegate storage.TypedAtomicStorage
}

func NewUserModelStorage(atomicStorage storage.AtomicStorage) *ModelUserStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/userModelStorage")
	userModelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelUserKey, reflect.TypeOf(ModelUserKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelUserData{}),
	)
	return &ModelUserStorage{delegate: userModelStorage}
}

func NewModelStorage(atomicStorage storage.AtomicStorage, orgMetadata *blockchain.OrganizationMetaData) *ModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/modelStorage")
	modelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelKey, reflect.TypeOf(ModelKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelData{}),
	)
	return &ModelStorage{delegate: modelStorage, organizationMetaData: orgMetadata}
}

func NewPendingModelStorage(atomicStorage storage.AtomicStorage) *PendingModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/pendingModelStorage")
	pendingModelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializePendingModelKey, reflect.TypeOf(PendingModelKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(PendingModelData{}),
	)
	return &PendingModelStorage{delegate: pendingModelStorage}
}

type ModelKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
	//GRPCMethodName  string
	//GRPCServiceName string
	ModelId string
}

func (key *ModelKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v|%v}", key.OrganizationId,
		key.ServiceId, key.GroupId, key.ModelId)
}

type ModelData struct {
	IsPublic            bool
	ModelName           string
	AuthorizedAddresses []string
	Status              Status
	CreatedByAddress    string
	ModelId             string
	UpdatedByAddress    string
	GroupId             string
	OrganizationId      string
	ServiceId           string
	GRPCMethodName      string
	GRPCServiceName     string
	Description         string
	IsDefault           bool
	TrainingLink        string
	UpdatedDate         string
	ValidatePrice       uint64
	TrainPrice          uint64
}

func (data *ModelData) String() string {
	return fmt.Sprintf("{DATA:%v|%v|%v|%v|%v|%v|IsPublic:%v|accesibleAddress:%v|createdBy:%v|updatedBy:%v|status:%v|TrainingLin:%v}",
		data.OrganizationId,
		data.ServiceId, data.GroupId, data.GRPCServiceName, data.GRPCMethodName, data.ModelId, data.AuthorizedAddresses, data.IsPublic,
		data.CreatedByAddress, data.UpdatedByAddress, data.Status, data.TrainingLink)
}

type ModelUserKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
	UserAddress    string
}

func (key *ModelUserKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v|%v}", key.OrganizationId,
		key.ServiceId, key.GroupId, key.UserAddress)
}

// ModelUserData maintain the list of all modelIds for a given user address
type ModelUserData struct {
	ModelIds []string
	//the below are only for display purposes
	OrganizationId string
	ServiceId      string
	GroupId        string
	//GRPCMethodName  string
	//GRPCServiceName string
	UserAddress string
}

func (data *ModelUserData) String() string {
	return fmt.Sprintf("{DATA:%v|%v|%v|%v|%v}",
		data.OrganizationId,
		data.ServiceId, data.GroupId, data.UserAddress, data.ModelIds)
}

type PendingModelKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
}

func (key *PendingModelKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v}", key.OrganizationId, key.ServiceId, key.GroupId)
}

type PendingModelData struct {
	ModelIDs []string
}

// PendingModelData maintain the list of all modelIds that have TRAINING\VALIDATING status
func (data *PendingModelData) String() string {
	return fmt.Sprintf("{DATA:%v}", data.ModelIDs)
}

func serializeModelKey(key any) (serialized string, err error) {
	modelKey := key.(*ModelKey)
	return modelKey.String(), nil
}

func (storage *ModelStorage) Get(key *ModelKey) (state *ModelData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*ModelData), ok, err
}

func (storage *ModelStorage) GetAll() (states []*ModelData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*ModelData), nil
}

func (storage *ModelStorage) Put(key *ModelKey, state *ModelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *ModelStorage) PutIfAbsent(key *ModelKey, state *ModelData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *ModelStorage) CompareAndSwap(key *ModelKey, prevState *ModelData,
	newState *ModelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

func (storage *ModelStorage) buildModelKey(modelID string) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        storage.organizationMetaData.GetGroupIdString(),
		ModelId:        modelID,
	}
	return
}

func (storage *ModelStorage) GetModel(modelID string) (data *ModelData, err error) {
	key := storage.buildModelKey(modelID)
	ok := false
	if data, ok, err = storage.Get(key); err != nil || !ok {
		zap.L().Warn("unable to retrieve model data from storage", zap.String("Model Id", key.ModelId), zap.Error(err))
	}
	return
}

func serializeModelUserKey(key any) (serialized string, err error) {
	modelUserKey := key.(*ModelUserKey)
	return modelUserKey.String(), nil
}

func (storage *ModelUserStorage) Get(key *ModelUserKey) (state *ModelUserData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*ModelUserData), ok, err
}

func (storage *ModelUserStorage) GetAll() (states []*ModelUserData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*ModelUserData), nil
}

func (storage *ModelUserStorage) Put(key *ModelUserKey, state *ModelUserData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *ModelUserStorage) PutIfAbsent(key *ModelUserKey, state *ModelUserData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *ModelUserStorage) CompareAndSwap(key *ModelUserKey, prevState *ModelUserData,
	newState *ModelUserData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

func serializePendingModelKey(key any) (serialized string, err error) {
	pendingModelKey := key.(*PendingModelKey)
	return pendingModelKey.String(), nil
}

func (storage *PendingModelStorage) Get(key *PendingModelKey) (state *PendingModelData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}

	return value.(*PendingModelData), ok, err
}

func (storage *PendingModelStorage) GetAll() (states []*PendingModelData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*PendingModelData), nil
}

func (storage *PendingModelStorage) Put(key *PendingModelKey, state *PendingModelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *PendingModelStorage) PutIfAbsent(key *PendingModelKey, state *PendingModelData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *PendingModelStorage) CompareAndSwap(key *PendingModelKey, prevState *PendingModelData,
	newState *PendingModelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}
