package escrow

import (
	"reflect"
)

// AtomicStorage is an interface to key-value storage with atomic operations.
type AtomicStorage interface {
	// Get returns value by key. ok value indicates whether passed key is
	// present in the storage. err indicates storage error.
	Get(key string) (value string, ok bool, err error)
	// GetByKeyPrefix returns list of values which keys has given prefix.
	GetByKeyPrefix(prefix string) (values []string, err error)
	// Put uncoditionally writes value by key in storage, err is not nil in
	// case of storage error.
	Put(key string, value string) (err error)
	// PutIfAbsent writes value if and only if key is absent in storage. ok is
	// true if key was absent and false otherwise. err indicates storage error.
	PutIfAbsent(key string, value string) (ok bool, err error)
	// CompareAndSwap atomically replaces prevValue by newValue. If ok flag is
	// true and err is nil then operation was successful. If err is nil and ok
	// is false then operation failed because prevValue is not equal to current
	// value. err indicates storage error.
	CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error)
	// Delete removes value by key
	Delete(key string) (err error)
}

// PrefixedAtomicStorage is decorator for atomic storage which adds a prefix to
// the storage keys.
type PrefixedAtomicStorage struct {
	delegate  AtomicStorage
	keyPrefix string
}

//It is recommended to use this function to create a PrefixedAtomicStorage
func NewPrefixedAtomicStorage(atomicStorage AtomicStorage, prefix string, mpeAddress string) *PrefixedAtomicStorage {
	return &PrefixedAtomicStorage{
		delegate:  atomicStorage,
		keyPrefix: "/" + mpeAddress + prefix,
	}
}

// Get is implementation of AtomicStorage.Get
func (storage *PrefixedAtomicStorage) Get(key string) (value string, ok bool, err error) {
	return storage.delegate.Get(storage.keyPrefix + "/" + key)
}

func (storage *PrefixedAtomicStorage) GetByKeyPrefix(prefix string) (values []string, err error) {
	return storage.delegate.GetByKeyPrefix(storage.keyPrefix + "/" + prefix)
}

// Put is implementation of AtomicStorage.Put
func (storage *PrefixedAtomicStorage) Put(key string, value string) (err error) {
	return storage.delegate.Put(storage.keyPrefix+"/"+key, value)
}

// PutIfAbsent is implementation of AtomicStorage.PutIfAbsent
func (storage *PrefixedAtomicStorage) PutIfAbsent(key string, value string) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(storage.keyPrefix+"/"+key, value)
}

// CompareAndSwap is implementation of AtomicStorage.CompareAndSwap
func (storage *PrefixedAtomicStorage) CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(storage.keyPrefix+"/"+key, prevValue, newValue)
}

func (storage *PrefixedAtomicStorage) Delete(key string) (err error) {
	return storage.delegate.Delete(storage.keyPrefix + "/" + key)
}

// TypedAtomicStorage is an atomic storage which automatically
// serializes/deserializes values and keys
type TypedAtomicStorage interface {
	// Get returns value by key
	Get(key interface{}) (value interface{}, ok bool, err error)
	// GetAll returns an array which contains all values from storage
	GetAll() (array interface{}, err error)
	// Put puts value by key unconditionally
	Put(key interface{}, value interface{}) (err error)
	// PutIfAbsent puts value by key if and only if key is absent in storage
	PutIfAbsent(key interface{}, value interface{}) (ok bool, err error)
	// CompareAndSwap puts newValue by key if and only if previous value is equal
	// to prevValue
	CompareAndSwap(key interface{}, prevValue interface{}, newValue interface{}) (ok bool, err error)
	// Delete removes value by key
	Delete(key interface{}) (err error)
}

// TypedAtomicStorageImpl is an implementation of TypedAtomicStorage interface
type TypedAtomicStorageImpl struct {
	atomicStorage     AtomicStorage
	keySerializer     func(key interface{}) (serialized string, err error)
	valueSerializer   func(value interface{}) (serialized string, err error)
	valueDeserializer func(serialized string, value interface{}) (err error)
	valueType         reflect.Type
}

// Get implements TypedAtomicStorage.Get
func (storage *TypedAtomicStorageImpl) Get(key interface{}) (value interface{}, ok bool, err error) {
	keyString, err := storage.keySerializer(key)
	if err != nil {
		return
	}

	valueString, ok, err := storage.atomicStorage.Get(keyString)
	if err != nil {
		return
	}
	if !ok {
		return
	}

	value = reflect.New(storage.valueType).Interface()
	err = storage.valueDeserializer(valueString, value)
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

func (storage *TypedAtomicStorageImpl) GetAll() (array interface{}, err error) {
	stringValues, err := storage.atomicStorage.GetByKeyPrefix("")
	if err != nil {
		return
	}

	values := reflect.MakeSlice(
		reflect.SliceOf(reflect.PtrTo(storage.valueType)),
		0, len(stringValues))

	for _, stringValue := range stringValues {
		value := reflect.New(storage.valueType)
		err = storage.valueDeserializer(stringValue, value.Interface())
		if err != nil {
			return nil, err
		}
		values = reflect.Append(values, value)
	}

	return values.Interface(), nil
}

// Put implementor TypedAtomicStorage.Put
func (storage *TypedAtomicStorageImpl) Put(key interface{}, value interface{}) (err error) {
	keyString, err := storage.keySerializer(key)
	if err != nil {
		return
	}

	valueString, err := storage.valueSerializer(value)
	if err != nil {
		return
	}

	return storage.atomicStorage.Put(keyString, valueString)
}

// PutIfAbsent implements TypedAtomicStorage.PutIfAbsent
func (storage *TypedAtomicStorageImpl) PutIfAbsent(key interface{}, value interface{}) (ok bool, err error) {
	keyString, err := storage.keySerializer(key)
	if err != nil {
		return
	}

	valueString, err := storage.valueSerializer(value)
	if err != nil {
		return
	}

	return storage.atomicStorage.PutIfAbsent(keyString, valueString)
}

// CompareAndSwap implements TypedAtomicStorage.CompareAndSwap
func (storage *TypedAtomicStorageImpl) CompareAndSwap(key interface{}, prevValue interface{}, newValue interface{}) (ok bool, err error) {
	keyString, err := storage.keySerializer(key)
	if err != nil {
		return
	}

	newValueString, err := storage.valueSerializer(newValue)
	if err != nil {
		return
	}

	prevValueString, err := storage.valueSerializer(prevValue)
	if err != nil {
		return
	}

	return storage.atomicStorage.CompareAndSwap(keyString, prevValueString, newValueString)
}

func (storage *TypedAtomicStorageImpl) Delete(key interface{}) (err error) {
	keyString, err := storage.keySerializer(key)
	if err != nil {
		return
	}

	return storage.atomicStorage.Delete(keyString)
}
