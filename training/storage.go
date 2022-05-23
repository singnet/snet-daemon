package training

import (
	"fmt"
	"github.com/singnet/snet-daemon/storage"
	"github.com/singnet/snet-daemon/utils"
	"reflect"
)

type ModelStorage struct {
	delegate storage.TypedAtomicStorage
}

func NewUserModelStorage(atomicStorage storage.AtomicStorage) *ModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelKey, reflect.TypeOf(ModelUserKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelUserData{}),
	)
	return &ModelStorage{delegate: storage}
}

type ModelUserKey struct {
	OrganizationId string
	ServiceId      string
	GroupID        string
	MethodName     string
	ModelId        string
}

func (key *ModelUserKey) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v/%v}", key.OrganizationId,
		key.ServiceId, key.GroupID, key.MethodName, key.ModelId)
}

func (data *ModelUserData) String() string {
	return fmt.Sprintf("{DATA:%v/%v/%v/%v/%v/isPublic:%v/accesibleAddress:%v/createdBy:%v/updatedBy:%v/status:%v}",
		data.OrganizationId,
		data.ServiceId, data.GroupId, data.MethodName, data.ModelId, data.AuthorizedAddresses, data.isPublic,
		data.CreatedByAddress, data.UpdatedByAddress, data.Status)
}

type ModelUserData struct {
	isPublic            bool
	AuthorizedAddresses []string
	Status              string
	CreatedByAddress    string
	ModelId             string
	UpdatedByAddress    string
	GroupId             string
	OrganizationId      string
	ServiceId           string
	MethodName          string
}

func serializeModelKey(key interface{}) (serialized string, err error) {
	myKey := key.(*ModelUserKey)
	return myKey.String(), nil
}
func (storage *ModelStorage) Get(key *ModelUserKey) (state *ModelUserData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*ModelUserData), ok, err
}

func (storage *ModelStorage) GetAll() (states []*ModelUserData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*ModelUserData), nil
}

func (storage *ModelStorage) Put(key *ModelUserKey, state *ModelUserData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *ModelStorage) PutIfAbsent(key *ModelUserKey, state *ModelUserData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *ModelStorage) CompareAndSwap(key *ModelUserKey, prevState *ModelUserData,
	newState *ModelUserData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}
