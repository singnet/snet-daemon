package freecall

import (
	"reflect"

	"github.com/singnet/snet-daemon/v6/storage"
)

// UserStorage persists free-call usage independently of MPE channels. Its legacy
// key format and gob field names are preserved for existing daemon instances.
type UserStorage struct {
	delegate storage.TypedAtomicStorage
}

// NewUserStorage adds the /free-call-user/storage prefix to the supplied backend.
// Daemon components supply organization/group/network-scoped atomic storage,
// without the MPE-address prefix used for payment-channel storage.
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
