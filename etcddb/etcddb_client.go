package etcddb

import (
	"context"
	"fmt"
	"time"

	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
)

const (
	// DefaultConnectionTimeout default connection timeout in milliseconds
	DefaultConnectionTimeout = 5000

	// DefaultRequestTimeout default request timeout in milliseconds
	DefaultRequestTimeout = 3000
)

// EtcdClient struct has some useful methods to wolrk with etcd client
type EtcdClient struct {
	timeout time.Duration
	etcdv3  *clientv3.Client
}

// NewEtcdClient create new etcd storage client.
func NewEtcdClient() (client *EtcdClient, err error) {
	return NewEtcdClientFromVip(config.Vip())
}

// NewEtcdClientFromVip create new etcd storage client from viper.
func NewEtcdClientFromVip(vip *viper.Viper) (client *EtcdClient, err error) {

	conf, err := GetPaymentChannelStorageClientConf(vip)

	if err != nil {
		return
	}

	log.WithField("PaymentChannelStorageClient", fmt.Sprintf("%+v", conf)).Info()

	connectionTimeout := time.Duration(conf.ConnectionTimeout) * time.Millisecond

	etcdv3, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Endpoints,
		DialTimeout: connectionTimeout,
	})

	if err != nil {
		return
	}

	requestTimeout := time.Duration(conf.RequestTimeout) * time.Millisecond
	client = &EtcdClient{timeout: requestTimeout, etcdv3: etcdv3}

	return
}

// Get gets value from etcd by key
func (client *EtcdClient) Get(key []byte) (value []byte, ok bool, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := client.etcdv3.Get(ctx, byteArraytoString(key))

	if err != nil {
		return
	}

	for _, kv := range response.Kvs {
		ok = true
		value = kv.Value
		return
	}

	return
}

// Put puts key and value to etcd
func (client *EtcdClient) Put(key []byte, value []byte) error {

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err := etcdv3.Put(ctx, byteArraytoString(key), byteArraytoString(value))

	return err
}

// Delete deletes the existing key and value from etcd
func (client *EtcdClient) Delete(key []byte) error {

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err := etcdv3.Delete(ctx, byteArraytoString(key))

	return err
}

// CompareAndSet uses CAS operation to set a value
func (client *EtcdClient) CompareAndSet(key []byte, expect []byte, update []byte) (bool, error) {
	if expect == nil {
		return client.createSync(key, update)
	} else {
		return client.compareAndSet(key, expect, update)
	}
}

func (client *EtcdClient) compareAndSet(key []byte, expect []byte, update []byte) (bool, error) {

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := etcdv3.KV.Txn(ctx).If(
		clientv3.Compare(clientv3.Value(byteArraytoString(key)), "=", byteArraytoString(expect)),
	).Then(
		clientv3.OpPut(byteArraytoString(key), byteArraytoString(update)),
	).Commit()

	if err != nil {
		return false, err
	}

	return response.Succeeded, nil
}

func (client *EtcdClient) createSync(key []byte, value []byte) (ok bool, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	etcdv3 := client.etcdv3
	session, err := concurrency.NewSession(etcdv3)

	if err != nil {
		return
	}

	lockKey := byteArraytoString(key)
	mu := concurrency.NewMutex(session, lockKey)
	err = mu.Lock(ctx)

	if err != nil {
		return
	}

	defer mu.Unlock(context.Background())

	response, err := etcdv3.Get(ctx, lockKey)

	if err != nil || response.Count != 0 {
		return
	}

	_, err = etcdv3.Put(ctx, lockKey, byteArraytoString(value))

	if err != nil {
		return
	}

	ok = true
	return
}

// Close closes etcd client
func (client *EtcdClient) Close() {
	client.etcdv3.Close()
}

func byteArraytoString(bytes []byte) string {
	return string(bytes)
}

func stringToByteArray(str string) []byte {
	return []byte(str)
}
