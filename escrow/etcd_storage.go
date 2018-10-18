package escrow

import (
	"bytes"
	"encoding/gob"

	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/spf13/viper"
)

// EtcdStorage storage based on etcd
type EtcdStorage struct {
	client *etcddb.EtcdClient
}

// NewEtcdStorage create an etcd storage
func NewEtcdStorage() (storage *EtcdStorage, err error) {
	return NewEtcdStorageFromVip(config.Vip())
}

// NewEtcdStorageFromVip create an etcd storage from Viper config
func NewEtcdStorageFromVip(vip *viper.Viper) (storage *EtcdStorage, err error) {
	client, err := etcddb.NewEtcdClientFromVip(vip)

	if err != nil {
		return
	}

	storage = &EtcdStorage{client: client}
	return
}

// Get gets value from etcd by key
func (storage *EtcdStorage) Get(key *PaymentChannelKey) (state *PaymentChannelData, ok bool, err error) {

	keyBytes, err := serialize(key)
	if err != nil {
		return
	}

	valueBytes, err := storage.client.Get(keyBytes)

	if err != nil || len(valueBytes) == 0 {
		return
	}

	state = &PaymentChannelData{}
	err = deserialize(valueBytes, state)

	if err != nil {
		return
	}

	ok = true
	return
}

// Put puts key and value to etcd
func (storage *EtcdStorage) Put(key *PaymentChannelKey, channel *PaymentChannelData) (err error) {

	keyBytes, err := serialize(key)
	if err != nil {
		return
	}

	valueBytes, err := serialize(channel)
	if err != nil {
		return
	}

	err = storage.client.Put(keyBytes, valueBytes)
	return
}

// CompareAndSwap uses CAS operation to set a value
func (storage *EtcdStorage) CompareAndSwap(
	key *PaymentChannelKey,
	prevState *PaymentChannelData,
	newState *PaymentChannelData,
) (ok bool, err error) {

	keyBytes, err := serialize(key)
	if err != nil {
		return
	}

	expectBytes, err := serialize(prevState)
	if err != nil {
		return
	}

	updateBytes, err := serialize(newState)
	if err != nil {
		return
	}

	ok, err = storage.client.CompareAndSet(keyBytes, expectBytes, updateBytes)
	return
}

// Close releases all retained resources
func (storage *EtcdStorage) Close() (err error) {
	storage.client.Close()
	return nil
}

func serialize(value interface{}) (slice []byte, err error) {

	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	err = e.Encode(value)
	if err != nil {
		return
	}

	slice = b.Bytes()
	return
}

func deserialize(slice []byte, value interface{}) (err error) {

	b := bytes.NewBuffer(slice)
	d := gob.NewDecoder(b)
	err = d.Decode(value)
	return
}
