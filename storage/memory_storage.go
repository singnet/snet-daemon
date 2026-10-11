package storage

import (
	"strings"
	"sync"
)

type MemoryStorage struct {
	data  map[string]string
	mutex *sync.RWMutex
}

// NewMemStorage returns a new in-memory atomic storage implementation
func NewMemStorage() (storage *MemoryStorage) {
	return &MemoryStorage{
		data:  make(map[string]string),
		mutex: &sync.RWMutex{},
	}
}

func (storage *MemoryStorage) Put(key, value string) (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	return storage.unsafePut(key, value)
}

func (storage *MemoryStorage) unsafePut(key, value string) (err error) {
	storage.data[key] = value
	return nil
}

func (storage *MemoryStorage) Get(key string) (value string, ok bool, err error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	return storage.unsafeGet(key)
}

func (storage *MemoryStorage) GetByKeyPrefix(prefix string) (values []string, err error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	for key, value := range storage.data {
		if strings.HasPrefix(key, prefix) {
			values = append(values, value)
		}
	}

	return
}

func (storage *MemoryStorage) unsafeGet(key string) (value string, ok bool, err error) {
	value, ok = storage.data[key]
	if !ok {
		return "", false, nil
	}
	return value, true, nil
}

func (storage *MemoryStorage) PutIfAbsent(key, value string) (ok bool, err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	_, ok, err = storage.unsafeGet(key)
	if err != nil {
		return
	}

	if ok {
		return false, nil
	}

	return true, storage.unsafePut(key, value)
}

func (storage *MemoryStorage) CompareAndSwap(key, prevValue, newValue string) (ok bool, err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	current, ok, err := storage.unsafeGet(key)
	if err != nil {
		return
	}

	if !ok || current != prevValue {
		return false, nil
	}

	return true, storage.unsafePut(key, newValue)
}

func (storage *MemoryStorage) Delete(key string) (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	delete(storage.data, key)

	return
}

func (storage *MemoryStorage) Clear() (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.data = make(map[string]string)

	return
}

func (storage *MemoryStorage) StartTransaction(conditionKeys []string) (transaction Transaction, err error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	conditionKeyValues := make([]KeyValueData, len(conditionKeys))
	for i, key := range conditionKeys {
		value, present := storage.data[key]
		conditionKeyValues[i] = KeyValueData{Key: key, Value: value, Present: present}
	}
	transaction = &memoryStorageTransaction{ConditionKeys: conditionKeys, ConditionValues: conditionKeyValues}
	return transaction, nil
}

func getValueDataForKey(key string, update []KeyValueData) (data KeyValueData, present bool) {
	for _, data := range update {
		if strings.Compare(data.Key, key) == 0 {
			return data, true
		}
	}
	return data, false
}

// CompleteTransaction checks all conditions and applies updates under one lock.
// A conflict returns false without applying any updates.
func (storage *MemoryStorage) CompleteTransaction(transaction Transaction, update []KeyValueData) (ok bool, err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	originalValues := transaction.(*memoryStorageTransaction).ConditionValues
	for _, oldData := range originalValues {
		currentValue, present := storage.data[oldData.Key]
		if present != oldData.Present || (present && currentValue != oldData.Value) {
			return false, nil
		}
	}
	// Preserve the existing behavior: only condition keys are updated.
	for _, oldData := range originalValues {
		if updatedData, found := getValueDataForKey(oldData.Key, update); found {
			storage.data[updatedData.Key] = updatedData.Value
		}
	}
	return true, nil
}

// ExecuteTransaction executes a transaction on the storage
func (storage *MemoryStorage) ExecuteTransaction(request CASRequest) (ok bool, err error) {
	maxRetries := 100
	for range maxRetries {
		// Every attempt needs a fresh snapshot; Update runs without holding the lock.
		transaction, err := storage.StartTransaction(request.ConditionKeys)
		if err != nil {
			return false, err
		}
		oldValues, err := transaction.GetConditionValues()
		if err != nil {
			return false, err
		}
		newValues, ok, err := request.Update(oldValues)
		if err != nil {
			return false, err
		}
		if !ok {
			// If the transaction was not successful and retrying is true - continue
			if request.RetryTillSuccessOrError {
				continue
			}
			return false, nil
		}
		ok, err = storage.CompleteTransaction(transaction, newValues)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if !request.RetryTillSuccessOrError {
			return false, nil
		}
	}
	// After exhausting retries, indicate failure without error to match expected semantics
	return false, nil
}

type memoryStorageTransaction struct {
	ConditionValues []KeyValueData
	ConditionKeys   []string
}

func (transaction *memoryStorageTransaction) GetConditionValues() ([]KeyValueData, error) {
	values := make([]KeyValueData, len(transaction.ConditionValues))
	for i, value := range transaction.ConditionValues {
		values[i] = KeyValueData{
			Key:     value.Key,
			Value:   value.Value,
			Present: value.Present,
		}
	}
	return values, nil
}
