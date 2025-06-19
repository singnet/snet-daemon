package escrow

import (
	"github.com/singnet/snet-daemon/v6/storage"
	"reflect"
)

type FreeCallUserStorage struct {
	delegate storage.TypedAtomicStorage
}

func NewFreeCallUserStorage(atomicStorage storage.AtomicStorage) *FreeCallUserStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/free-call-user/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeFreeCallKey, reflect.TypeOf(FreeCallUserKey{}), serialize, deserialize,
		reflect.TypeOf(FreeCallUserData{}),
	)
	return &FreeCallUserStorage{delegate: storage}
	/*	return &FreeCallUserStorage{
		delegate: &storage.TypedAtomicStorageImpl{
			atomicStorage: &storage.PrefixedAtomicStorage{
				delegate:  atomicStorage,
				keyPrefix: "/free-call-user/storage",
			},
			keySerializer:     serializeFreeCallKey,
			keyType:           reflect.TypeOf(FreeCallUserKey{}),
			valueSerializer:   serialize,
			valueDeserializer: deserialize,
			valueType:         reflect.TypeOf(FreeCallUserData{}),
		},
	}*/
}

func serializeFreeCallKey(key any) (serialized string, err error) {
	myKey := key.(*FreeCallUserKey)
	return myKey.String(), nil
}

func (storage *FreeCallUserStorage) Get(key *FreeCallUserKey) (state *FreeCallUserData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*FreeCallUserData), ok, err
}

func (storage *FreeCallUserStorage) GetAll() (states []*FreeCallUserData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*FreeCallUserData), nil
}

func (storage *FreeCallUserStorage) Put(key *FreeCallUserKey, state *FreeCallUserData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *FreeCallUserStorage) PutIfAbsent(key *FreeCallUserKey, state *FreeCallUserData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *FreeCallUserStorage) CompareAndSwap(key *FreeCallUserKey, prevState *FreeCallUserData, newState *FreeCallUserData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}
