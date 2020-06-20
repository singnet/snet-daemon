package escrow

import (
	"fmt"
	"strings"
	"sync"
)

type memoryStorage struct {
	data  map[string]string
	mutex *sync.RWMutex
}

// NewMemStorage returns new in-memory atomic storage implementation
func NewMemStorage() (storage *memoryStorage) {
	return &memoryStorage{
		data:  make(map[string]string),
		mutex: &sync.RWMutex{},
	}
}

func (storage *memoryStorage) Put(key, value string) (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	return storage.unsafePut(key, value)
}

func (storage *memoryStorage) unsafePut(key, value string) (err error) {
	storage.data[key] = value
	return nil
}

func (storage *memoryStorage) Get(key string) (value string, ok bool, err error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	return storage.unsafeGet(key)
}

func (storage *memoryStorage) GetByKeyPrefix(prefix string) (values []string, err error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	for key, value := range storage.data {
		if strings.HasPrefix(key, prefix) {
			values = append(values, value)
		}
	}

	return
}

func (storage *memoryStorage) unsafeGet(key string) (value string, ok bool, err error) {
	value, ok = storage.data[key]
	if !ok {
		return "", false, nil
	}
	return value, true, nil
}

func (storage *memoryStorage) PutIfAbsent(key, value string) (ok bool, err error) {
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

func (storage *memoryStorage) CompareAndSwap(key, prevValue, newValue string) (ok bool, err error) {
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

func (storage *memoryStorage) Delete(key string) (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	delete(storage.data, key)

	return
}

func (storage *memoryStorage) Clear() (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	storage.data = make(map[string]string)

	return
}

func (storage *memoryStorage) StartTransaction(keyPrefix string) (transaction Transaction, err error) {
	return nil, fmt.Errorf("Not implemented")
}

func (storage *memoryStorage) CompleteTransaction(transaction Transaction, update []*KeyValueData) (ok bool, err error) {
	return false, fmt.Errorf("Not implemented")
}
