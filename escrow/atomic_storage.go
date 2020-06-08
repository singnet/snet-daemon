package escrow

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
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
	CAS(request *CASRequest) (response *CASResponse, err error)
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

func (storage *PrefixedAtomicStorage) CAS(request *CASRequest) (response *CASResponse, err error) {
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
	// CompareAndSwap puts newValues by key if and only if previous value is equal
	// to prevValue
	CompareAndSwap(key interface{}, prevValue interface{}, newValue interface{}) (ok bool, err error)
	// Delete removes value by key
	Delete(key interface{}) (err error)
}

type ReadFunc func(params ...interface{}) (businessData interface{}, err error)
type ConditionFunc func(params ...interface{}) (newValues interface{}, err error)
type ActionFunc func(params ...interface{}) (casOldValues []*KeyValueData, casNewValues []*KeyValueData, err error)

type CASRequest struct {
	KeyPrefix    string
	OldKeyValues []*KeyValueData
	NewKeyValues []*KeyValueData
	Read         ReadFunc
	Condition    ConditionFunc
	Action       ActionFunc
}

type CASResponse struct {
	Succeeded  bool
	LatestData []*KeyValueData
}
type KeyValueData struct {
	//Incase we need to switch to revision comparision, this is much faster than value comparision
	Version int64
	Value   string
	Key     string //Lets not keep it over generic and complicate, make the key as string
	Compare CustomCompareOptions
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

func (storage *TypedAtomicStorageImpl) VerifyAndUpdate(key string) (err error) {

	return nil
}

var (
	ConvertRawDataToPrePaidUsage = func(params ...interface{}) (new interface{}, err error) {
		businessObject := &PrePaidUsageData{}

		compare := CustomCompareOptions{Operator: EQUAL, CompareOn: MODIFIED_VERSION}
		data := params[0].([]*KeyValueData)
		for _, dataRetrieved := range data {
			if dataRetrieved == nil {
				continue
			}
			dataRetrieved.Compare = compare
			amount := big.NewInt(0)
			if _, ok := amount.SetString(dataRetrieved.Value, 10); !ok {
				return nil, fmt.Errorf("Unable to convert %v to BigInt, key:%v",
					dataRetrieved.Value, dataRetrieved.Key)
			}
			//todo for now hardcoding the channel Id, remove this
			businessObject.ChannelID = big.NewInt(1)
			//Lets try to use serialize and de serialize here todo rather than this !
			businessObject.LastModifiedVersion = dataRetrieved.Version
			if strings.Contains(dataRetrieved.Key, USED_AMOUNT) {
				businessObject.UsedAmount = amount
				businessObject.UsageType = USED_AMOUNT
			} else if strings.Contains(dataRetrieved.Key, PLANNED_AMOUNT) {
				businessObject.PlannedAmount = amount
				businessObject.UsageType = PLANNED_AMOUNT
			} else if strings.Contains(dataRetrieved.Key, FAILED_AMOUNT) {
				businessObject.FailedAmount = amount
				businessObject.UsageType = FAILED_AMOUNT
			}
		}
		if businessObject.PlannedAmount == nil {
			return nil, fmt.Errorf("Planned amount cannot be Nil")
		}
		if businessObject.FailedAmount == nil {
			businessObject.FailedAmount = big.NewInt(0)
		}
		if businessObject.UsedAmount == nil {
			businessObject.UsedAmount = big.NewInt(0)
		}
		return businessObject, nil
	}

	BuildOldAndNewValuesForCAS = func(params ...interface{}) (oldValues []*KeyValueData,
		newValues []*KeyValueData, err error) {
		if len(params) == 0 {
			return nil, nil, fmt.Errorf("No parameters passed for the Action function")
		}
		data := params[0].(*PrePaidUsageData)
		if data == nil {
			return nil, nil, fmt.Errorf("Expected PrePaidUsageData in Params as the first parmeter")
		}
		newValue := &KeyValueData{Key: data.Key(), Value: data.UsedAmount.String()}
		newValues = make([]*KeyValueData, 0)
		oldValues = make([]*KeyValueData, 0)
		oldValue := &KeyValueData{
			Key:     data.Key(),
			Version: data.LastModifiedVersion,
			Compare: CustomCompareOptions{Operator: EQUAL, CompareOn: MODIFIED_VERSION},
		}
		newValues = append(newValues, newValue)
		oldValues = append(oldValues, oldValue)

		return oldValues, newValues, nil
	}
)

var (
	IncrementUsageAmount = func(params ...interface{}) (new interface{}, err error) {

		if len(params) == 0 {
			return nil, fmt.Errorf("You need to pass a struct of type PrePaidUsageData")
		}
		//Make sure the order expected is honored
		//create a dynamic function and initialize the price from there , this way
		//you dont need to pass the price !!!!!!!! todo anonymous function

		price := big.NewInt(3)
		oldState := params[0].(*PrePaidUsageData)
		newState := oldState.Clone()
		newState.UsedAmount.Add(price, oldState.UsedAmount)
		if newState.UsedAmount.Cmp(oldState.PlannedAmount.Add(oldState.PlannedAmount, oldState.FailedAmount)) > 0 {
			return nil, fmt.Errorf("Usage Exceeded on channel %v", oldState.ChannelID)
		}
		return newState, nil

	}
)
