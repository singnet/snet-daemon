package freecall

import (
	"reflect"

	"github.com/singnet/snet-daemon/v6/storage"
)

type UserStorage struct {
	delegate storage.TypedAtomicStorage
}

func NewUserStorage(atomicStorage storage.AtomicStorage) *UserStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/free-call-user/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeFreeCallKey, reflect.TypeFor[UserKey](), storage.EncodeGob, storage.DecodeGob,
		reflect.TypeFor[UserData](),
	)
	return &UserStorage{delegate: storage}
}

func serializeFreeCallKey(key any) (serialized string, err error) {
	myKey := key.(*UserKey)
	return myKey.String(), nil
}

func (storage *UserStorage) Get(key *UserKey) (state *UserData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*UserData), ok, err
}

func (storage *UserStorage) GetAll() (states []*UserData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*UserData), nil
}

func (storage *UserStorage) Put(key *UserKey, state *UserData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *UserStorage) PutIfAbsent(key *UserKey, state *UserData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *UserStorage) CompareAndSwap(key *UserKey, prevState *UserData, newState *UserData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}
