package storage

import (
	"reflect"
	"strings"
)

// AtomicStorage is an interface to key-value storage with atomic operations.
type AtomicStorage interface {
	// Get returns value by key. Ok value indicates whether the passed key is
	// present in the storage. Err indicates a storage error.
	Get(key string) (value string, ok bool, err error)
	// GetByKeyPrefix returns a list of values which keys have given prefix.
	GetByKeyPrefix(prefix string) (values []string, err error)
	// Put unconditionally writes value by key in storage, err is not nil in
	// case of storage error.
	Put(key string, value string) (err error)
	// PutIfAbsent writes value if and only if the key is absent in storage. ok is
	// true if the key was absent and false otherwise. err indicates a storage error.
	PutIfAbsent(key string, value string) (ok bool, err error)
	// CompareAndSwap atomically replaces prevValue by newValue. If an ok flag is
	// true and err is nil, then the operation was successful. If err is nil and ok
	// is false, then the operation failed because prevValue is not equal to the current
	// value. err indicates a storage error.
	CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error)
	// Delete removes value by key
	Delete(key string) (err error)

	StartTransaction(conditionKeys []string) (transaction Transaction, err error)
	CompleteTransaction(transaction Transaction, update []KeyValueData) (ok bool, err error)
	ExecuteTransaction(request CASRequest) (ok bool, err error)
}

type Transaction interface {
	GetConditionValues() ([]KeyValueData, error)
}

type UpdateFunc func(conditionValues []KeyValueData) (update []KeyValueData, ok bool, err error)

type CASRequest struct {
	RetryTillSuccessOrError bool
	Update                  UpdateFunc
	ConditionKeys           []string
}

type KeyValueData struct {
	Key     string
	Value   string
	Present bool
}

// PrefixedAtomicStorage is a decorator for atomic storage that adds a prefix to
// the storage keys.
type PrefixedAtomicStorage struct {
	delegate  AtomicStorage
	keyPrefix string
}

// NewPrefixedAtomicStorage use this function to create a PrefixedAtomicStorage
func NewPrefixedAtomicStorage(atomicStorage AtomicStorage, prefix string) *PrefixedAtomicStorage {
	return &PrefixedAtomicStorage{
		delegate:  atomicStorage,
		keyPrefix: prefix,
	}
}

// Get is implementation of AtomicStorage.Get
func (storage *PrefixedAtomicStorage) Get(key string) (value string, ok bool, err error) {
	return storage.delegate.Get(storage.keyPrefix + "/" + key)
}

func (storage *PrefixedAtomicStorage) GetByKeyPrefix(prefix string) (values []string, err error) {
	return storage.delegate.GetByKeyPrefix(storage.keyPrefix + "/" + prefix)
}

// Put is an implementation of AtomicStorage.Put
func (storage *PrefixedAtomicStorage) Put(key string, value string) (err error) {
	return storage.delegate.Put(storage.keyPrefix+"/"+key, value)
}

// PutIfAbsent is an implementation of AtomicStorage.PutIfAbsent
func (storage *PrefixedAtomicStorage) PutIfAbsent(key string, value string) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(storage.keyPrefix+"/"+key, value)
}

// CompareAndSwap is an implementation of AtomicStorage.CompareAndSwap
func (storage *PrefixedAtomicStorage) CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(storage.keyPrefix+"/"+key, prevValue, newValue)
}

func (storage *PrefixedAtomicStorage) Delete(key string) (err error) {
	return storage.delegate.Delete(storage.keyPrefix + "/" + key)
}

type prefixedTransactionImpl struct {
	transaction Transaction
	storage     *PrefixedAtomicStorage
}

// Compile-time check: prefixedTransactionImpl implements Transaction.
var _ Transaction = (*prefixedTransactionImpl)(nil)

func (transaction *prefixedTransactionImpl) GetConditionValues() ([]KeyValueData, error) {
	conditionKeyValues, err := transaction.transaction.GetConditionValues()
	if err != nil {
		return nil, err
	}
	unPrefixedKeyValues := transaction.storage.removeKeyValuePrefix(conditionKeyValues)
	return unPrefixedKeyValues, nil
}

func (storage *PrefixedAtomicStorage) StartTransaction(conditionKeys []string) (transaction Transaction, err error) {
	prefixedKeys := storage.appendKeyPrefix(conditionKeys)
	transaction, err = storage.delegate.StartTransaction(prefixedKeys)
	if err != nil {
		return nil, err
	}
	return &prefixedTransactionImpl{storage: storage, transaction: transaction}, nil
}

func (storage *PrefixedAtomicStorage) appendKeyPrefix(conditionKeys []string) (preFixedConditionKeys []string) {
	prefixedKeys := make([]string, len(conditionKeys))
	copy(prefixedKeys, conditionKeys)
	for i, key := range prefixedKeys {
		prefixedKeys[i] = storage.keyPrefix + "/" + key
	}
	return prefixedKeys
}

func (storage *PrefixedAtomicStorage) appendKeyValuePrefix(update []KeyValueData) []KeyValueData {
	if update == nil {
		return nil
	}
	prefixedKeyValueData := make([]KeyValueData, len(update))
	copy(prefixedKeyValueData, update)
	for i, keyValue := range prefixedKeyValueData {
		prefixedKeyValueData[i].Key = storage.keyPrefix + "/" + keyValue.Key
	}
	return prefixedKeyValueData
}

func (storage *PrefixedAtomicStorage) removeKeyValuePrefix(in []KeyValueData) []KeyValueData {
	out := make([]KeyValueData, len(in))
	copy(out, in)
	p := strings.TrimRight(storage.keyPrefix, "/") + "/"
	for i := range out {
		out[i].Key = strings.TrimPrefix(out[i].Key, p)
	}
	return out
}

func (storage *PrefixedAtomicStorage) CompleteTransaction(transaction Transaction, update []KeyValueData) (ok bool, err error) {
	return storage.delegate.CompleteTransaction(transaction.(*prefixedTransactionImpl).transaction, storage.appendKeyValuePrefix(update))
}

func (storage *PrefixedAtomicStorage) ExecuteTransaction(request CASRequest) (ok bool, err error) {
	updateFunction := func(conditionKeyValues []KeyValueData) (update []KeyValueData, ok bool, err error) {
		//the keys retrieved will have the storage prefix, we need to remove it! else deserialize of a key will fail
		originalKeyValues := storage.removeKeyValuePrefix(conditionKeyValues)
		newValues, ok, err := request.Update(originalKeyValues)
		return storage.appendKeyValuePrefix(newValues), ok, err
	}
	prefixedRequest := CASRequest{
		ConditionKeys:           storage.appendKeyPrefix(request.ConditionKeys),
		RetryTillSuccessOrError: request.RetryTillSuccessOrError,
		Update:                  updateFunction,
	}
	return storage.delegate.ExecuteTransaction(prefixedRequest)
}

// TypedAtomicStorage is an atomic storage that automatically
// serializes/deserializes values and keys
type TypedAtomicStorage interface {
	// Get return value by key
	Get(key any) (value any, ok bool, err error)
	// GetAll returns an array which contains all values from storage
	GetAll() (array any, err error)
	// Put puts value by key unconditionally
	Put(key any, value any) (err error)
	// PutIfAbsent puts value by key if and only if the key is absent in storage
	PutIfAbsent(key any, value any) (ok bool, err error)
	// CompareAndSwap puts newValue by key if and only if the previous value is equal
	// to prevValue
	CompareAndSwap(key any, prevValue any, newValue any) (ok bool, err error)
	// Delete removes value by key
	Delete(key any) (err error)
	ExecuteTransaction(request TypedCASRequest) (ok bool, err error)
}

type TypedTransaction interface {
	GetConditionValues() ([]TypedKeyValueData, error)
}

// Best to change this to KeyValueData, will do this in the next commit
type TypedUpdateFunc func(conditionValues []TypedKeyValueData) (update []TypedKeyValueData, ok bool, err error)

type TypedCASRequest struct {
	RetryTillSuccessOrError bool
	Update                  TypedUpdateFunc
	ConditionKeys           []any //Typed Keys
}

type TypedKeyValueData struct {
	Key     any
	Value   any
	Present bool
}

// TypedAtomicStorageImpl is an implementation of TypedAtomicStorage interface
type TypedAtomicStorageImpl struct {
	atomicStorage     AtomicStorage
	keySerializer     func(key any) (serialized string, err error)
	keyType           reflect.Type
	valueSerializer   func(value any) (serialized string, err error)
	valueDeserializer func(serialized string, value any) (err error)
	valueType         reflect.Type
}

func NewTypedAtomicStorageImpl(storage AtomicStorage, keySerializer func(key any) (serialized string, err error),
	keyType reflect.Type, valueSerializer func(value any) (serialized string, err error),
	valueDeserializer func(serialized string, value any) (err error),
	valueType reflect.Type) TypedAtomicStorage {
	return &TypedAtomicStorageImpl{
		atomicStorage:     storage,
		keySerializer:     keySerializer,
		keyType:           keyType,
		valueSerializer:   valueSerializer,
		valueDeserializer: valueDeserializer,
		valueType:         valueType,
	}
}

// Get implements TypedAtomicStorage.Get
func (storage *TypedAtomicStorageImpl) Get(key any) (value any, ok bool, err error) {
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

	value, err = storage.deserializeValue(valueString)
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

func (storage *TypedAtomicStorageImpl) deserializeValue(valueString string) (value any, err error) {
	value = reflect.New(storage.valueType).Interface()
	err = storage.valueDeserializer(valueString, value)
	if err != nil {
		return nil, err
	}
	return value, err
}

func (storage *TypedAtomicStorageImpl) GetAll() (array any, err error) {
	stringValues, err := storage.atomicStorage.GetByKeyPrefix("")
	if err != nil {
		return
	}

	values := reflect.MakeSlice(
		reflect.SliceOf(reflect.PointerTo(storage.valueType)),
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

// Put implementer TypedAtomicStorage.Put
func (storage *TypedAtomicStorageImpl) Put(key any, value any) (err error) {
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
func (storage *TypedAtomicStorageImpl) PutIfAbsent(key any, value any) (ok bool, err error) {
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
func (storage *TypedAtomicStorageImpl) CompareAndSwap(key any, prevValue any, newValue any) (ok bool, err error) {
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

func (storage *TypedAtomicStorageImpl) Delete(key any) (err error) {
	keyString, err := storage.keySerializer(key)
	if err != nil {
		return
	}

	return storage.atomicStorage.Delete(keyString)
}

func (storage *TypedAtomicStorageImpl) convertKeyValueDataToTyped(conditionKeys []any, keyValueData []KeyValueData) (result []TypedKeyValueData, err error) {
	result = make([]TypedKeyValueData, len(conditionKeys))

	for i, conditionKey := range conditionKeys {
		conditionKeyString, err := storage.keySerializer(conditionKey)
		if err != nil {
			return nil, err
		}
		result[i] = TypedKeyValueData{
			Key:     conditionKey,
			Present: false,
		}

		keyValueString, ok := findKeyValueByKey(keyValueData, conditionKeyString)
		if !ok || !keyValueString.Present {
			// Either key wasn't found or value marked absent — skip deserialization
			continue
		}
		result[i].Present = true
		result[i].Value, err = storage.deserializeValue(keyValueString.Value)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func findKeyValueByKey(data []KeyValueData, key string) (*KeyValueData, bool) {
	for i := range data {
		if data[i].Key == key {
			return &data[i], true
		}
	}
	return nil, false
}

func (storage *TypedAtomicStorageImpl) ExecuteTransaction(request TypedCASRequest) (ok bool, err error) {
	updateFunction := func(conditionValues []KeyValueData) (update []KeyValueData, ok bool, err error) {
		typedValues, err := storage.convertKeyValueDataToTyped(request.ConditionKeys, conditionValues)
		if err != nil {
			return nil, false, err
		}
		typedUpdate, ok, err := request.Update(typedValues)
		if err != nil {
			return nil, ok, err
		}
		stringUpdate, err := storage.convertTypedKeyValueDataToString(typedUpdate)
		return stringUpdate, ok, err
	}
	conditionKeysString, err := storage.convertTypedKeyToString(request.ConditionKeys)
	if err != nil {
		return false, err
	}
	storageRequest := CASRequest{
		RetryTillSuccessOrError: request.RetryTillSuccessOrError,
		Update:                  updateFunction,
		ConditionKeys:           conditionKeysString,
	}
	return storage.atomicStorage.ExecuteTransaction(storageRequest)
}

func (storage *TypedAtomicStorageImpl) convertTypedKeyToString(typedKeys []any) (stringKeys []string, err error) {
	stringKeys = make([]string, len(typedKeys))
	for i, key := range typedKeys {
		stringKeys[i], err = storage.keySerializer(key)
		if err != nil {
			return nil, err
		}
	}
	return stringKeys, nil
}

func (storage *TypedAtomicStorageImpl) convertTypedKeyValueDataToString(
	update []TypedKeyValueData) (data []KeyValueData, err error) {
	if update == nil {
		return nil, nil
	}
	updateString := make([]KeyValueData, len(update))
	for i, keyValue := range update {
		updateString[i] = KeyValueData{
			Present: keyValue.Present,
		}
		updateString[i].Key, err = storage.keySerializer(keyValue.Key)
		if err != nil {
			return nil, err
		}
		if !keyValue.Present {
			continue
		}
		updateString[i].Value, err = storage.valueSerializer(keyValue.Value)
		if err != nil {
			return nil, err
		}
	}
	return updateString, nil
}
