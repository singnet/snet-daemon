package escrow

import (
	"fmt"
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
	//Compares and Swaps if the compare conditions are met , else it returns you the
	//Latest version and Value of the key passed , this can be used to do any pre Validations and request
	//again for an update
	CAS(request *CASRequestEtcd) (response *CASResponseEtcd, err error)
}

// PrefixedAtomicStorage is decorator for atomic storage which adds a prefix to
// the storage keys.
type PrefixedAtomicStorage struct {
	delegate  AtomicStorage
	keyPrefix string
}

//It is recommended to use this function to create a PrefixedAtomicStorage
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

func (storage *PrefixedAtomicStorage) CAS(request *CASRequestEtcd) (response *CASResponseEtcd, err error) {
	return storage.delegate.CAS(request)
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

	VerifyAndUpdate(details *ValidateAndUpdateStorageDetails) (err error)
}

type CASRequestEtcd struct {
	Key             string
	OldValue        interface{}
	ModifiedVersion int64
	NewValue        string
	Compare         CustomCompareOptions
}

type CASResponseEtcd struct {
	Key             string
	Value           string
	ModifiedVersion int64
	Succeeded       bool
}

type CustomCompareOptions struct {
	Operator  Comparison_Operator
	CompareOn Comparison_On
}
type Comparison_On string

const (
	VALUE            Comparison_On = "Value"
	MODIFIED_VERSION Comparison_On = "Modified_Version"
)

type Comparison_Operator string

const (
	EQUAL     Comparison_Operator = "="
	GREATER   Comparison_Operator = ">"
	LESS      Comparison_Operator = "<"
	NOT_EQUAL Comparison_Operator = "!="
)

type ValidateAndUpdateStorageDetails struct {
	//old Value of the record
	oldValue interface{}
	//new value to be replaced with
	newValue interface{}

	//Consider the following Scenarios

	/*
			oldvalue is the state you last read from the storage
			newValue is the state you decide based on some validations and derive this state
			Just before you update , you need to make sure the oldvalue you got was actually the latest value
			if it was not, then you need to take the latest value from storage and apply any validations
			and then re generate the newValue, the below function helps you do this when the retry option is set to true.
		    if you dont want the old value to be swapped with the latest DB value
	*/
	//this function will take the old value , check for conditions and return back a new Value.

	Validate func(oldValue interface{}, params ...interface{}) (newValue interface{}, err error)
	Retry    bool
	//to retrieve the record
	Key interface{}
	//What kind of comparision is desired
	Comparision Comparison_Operator
	Params      interface{}
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

func (storage *TypedAtomicStorageImpl) VerifyAndUpdate(details *ValidateAndUpdateStorageDetails) (err error) {

	oldValue := details.oldValue
	newValue := details.newValue
	//oldValueString, err := serialize(oldValue)
	key, err := serialize(details.Key)
	var newValueString string
	request := &CASRequestEtcd{
		Key:             key,
		ModifiedVersion: 1, //You cannot help it , we need to make one Read to get the latest version or value
		Compare:         CustomCompareOptions{Operator: EQUAL, CompareOn: MODIFIED_VERSION},
	}
	//todo, how long do we re try, disucss this with Team ?
	for {
		if newValue != nil {
			if newValueString, err = serialize(newValue); err != nil {
				return err
			}
			request.NewValue = newValueString
		}

		response, err := storage.atomicStorage.CAS(request)
		if err != nil {
			return err
		}
		//Nothing to do more, the comparision and update was successful
		if response.Succeeded {
			return nil
		}

		if !details.Retry {
			return fmt.Errorf("CAS did not succeed ,retry not attempted")
		}
		oldValue = reflect.New(storage.valueType).Interface()
		err = deserialize(string(response.Value), oldValue)
		request.ModifiedVersion = response.ModifiedVersion
		newValue, err = details.Validate(oldValue, details.Params)
		if err != nil {
			return err
		}

	}
}
