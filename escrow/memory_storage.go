package escrow

import (
	"bytes"
	"encoding/gob"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
)

type memoryStorageKey string

type memoryStorage struct {
	data  map[memoryStorageKey]*PaymentChannelData
	mutex *sync.RWMutex
}

func NewMemStorage() (storage PaymentChannelStorage) {
	return &memoryStorage{
		data:  make(map[memoryStorageKey]*PaymentChannelData),
		mutex: &sync.RWMutex{},
	}
}

func getMemoryStorageKey(key *PaymentChannelKey) memoryStorageKey {
	return memoryStorageKey(fmt.Sprintf("%v", key))
}

func (storage *memoryStorage) Put(key *PaymentChannelKey, channel *PaymentChannelData) (err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	return storage.unsafePut(key, channel)
}

func (storage *memoryStorage) unsafePut(key *PaymentChannelKey, channel *PaymentChannelData) (err error) {
	storage.data[getMemoryStorageKey(key)] = channel
	return nil
}

func (storage *memoryStorage) Get(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	return storage.unsafeGet(key)
}

func (storage *memoryStorage) unsafeGet(key *PaymentChannelKey) (channel *PaymentChannelData, ok bool, err error) {
	channel, ok = storage.data[getMemoryStorageKey(key)]
	if !ok {
		return nil, false, nil
	}
	return channel, true, nil
}

func (storage *memoryStorage) CompareAndSwap(key *PaymentChannelKey, prevState *PaymentChannelData, newState *PaymentChannelData) (ok bool, err error) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	current, ok, err := storage.unsafeGet(key)
	if err != nil {
		return
	}
	if prevState == nil {
		if ok {
			return false, nil
		}
	} else {
		if !ok {
			return false, nil
		}
		if !bytes.Equal(toBytes(current), toBytes(prevState)) {
			return false, nil
		}
	}
	return true, storage.unsafePut(key, newState)
}

func toBytes(data interface{}) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(data)
	if err != nil {
		log.WithError(err).Fatal("Error while encoding value to binary")
	}
	return buffer.Bytes()
}
