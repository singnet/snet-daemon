package training

import (
	"fmt"
	"github.com/singnet/snet-daemon/v5/storage"
	"github.com/singnet/snet-daemon/v5/utils"
	"reflect"
)

type ModelStorage struct {
	delegate storage.TypedAtomicStorage
}
type ModelUserStorage struct {
	delegate storage.TypedAtomicStorage
}

func NewUerModelStorage(atomicStorage storage.AtomicStorage) *ModelUserStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/userModelStorage")
	userModelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelUserKey, reflect.TypeOf(ModelUserKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelUserData{}),
	)
	return &ModelUserStorage{delegate: userModelStorage}
}

func NewModelStorage(atomicStorage storage.AtomicStorage) *ModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/modelStorage")
	modelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelKey, reflect.TypeOf(ModelKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelData{}),
	)
	return &ModelStorage{delegate: modelStorage}
}

type ModelUserKey struct {
	OrganizationId  string
	ServiceId       string
	GroupId         string
	GRPCMethodName  string
	GRPCServiceName string
	UserAddress     string
}

func (key *ModelUserKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v|%v|%v|%v}", key.OrganizationId,
		key.ServiceId, key.GroupId, key.GRPCServiceName, key.GRPCMethodName, key.UserAddress)
}

// ModelUserData maintain the list of all modelIds for a given user address
type ModelUserData struct {
	ModelIds []string
	//the below are only for display purposes
	OrganizationId  string
	ServiceId       string
	GroupId         string
	GRPCMethodName  string
	GRPCServiceName string
	UserAddress     string
}

func (data *ModelUserData) String() string {
	return fmt.Sprintf("{DATA:%v|%v|%v|%v|%v|%v|%v}",
		data.OrganizationId,
		data.ServiceId, data.GroupId, data.GRPCMethodName, data.GRPCServiceName, data.UserAddress, data.ModelIds)
}

type ModelKey struct {
	OrganizationId  string
	ServiceId       string
	GroupId         string
	GRPCMethodName  string
	GRPCServiceName string
	ModelId         string
}

func (key *ModelKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v|%v|%v|%v}", key.OrganizationId,
		key.ServiceId, key.GroupId, key.GRPCServiceName, key.GRPCMethodName, key.ModelId)
}

func (data *ModelData) String() string {
	return fmt.Sprintf("{DATA:%v|%v|%v|%v|%v|%v|IsPublic:%v|accesibleAddress:%v|createdBy:%v|updatedBy:%v|status:%v|TrainingLin:%v}",
		data.OrganizationId,
		data.ServiceId, data.GroupId, data.GRPCServiceName, data.GRPCMethodName, data.ModelId, data.AuthorizedAddresses, data.IsPublic,
		data.CreatedByAddress, data.UpdatedByAddress, data.Status, data.TrainingLink)
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
}

func serializeModelKey(key any) (serialized string, err error) {
	myKey := key.(*ModelKey)
	return myKey.String(), nil
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
func serializeModelUserKey(key any) (serialized string, err error) {
	myKey := key.(*ModelUserKey)
	return myKey.String(), nil
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
