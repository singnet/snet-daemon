package escrow

import (
	"sync"
)

type memoryStorage struct {
	data  map[string]string
	mutex *sync.RWMutex
}

func NewMemStorage() (storage AtomicStorage) {
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
