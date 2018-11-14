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

// EtcdClientMutex mutex struct for etcd client
type EtcdClientMutex struct {
	mutex *concurrency.Mutex
}

// Lock lock etcd key
func (mutex *EtcdClientMutex) Lock(ctx context.Context) (err error) {
	return mutex.mutex.Lock(ctx)
}

// Unlock unlock etcd key
func (mutex *EtcdClientMutex) Unlock(ctx context.Context) (err error) {
	return mutex.mutex.Unlock(ctx)
}

// EtcdClient struct has some useful methods to wolrk with etcd client
type EtcdClient struct {
	timeout time.Duration
	session *concurrency.Session
	etcdv3  *clientv3.Client
}

// NewEtcdClient create new etcd storage client.
func NewEtcdClient() (client *EtcdClient, err error) {
	return NewEtcdClientFromVip(config.Vip())
}

// NewEtcdClientFromVip create new etcd storage client from viper.
func NewEtcdClientFromVip(vip *viper.Viper) (client *EtcdClient, err error) {

	conf, err := GetEtcdClientConf(vip)

	if err != nil {
		return
	}

	log.WithField("PaymentChannelStorageClient", fmt.Sprintf("%+v", conf)).Info()

	etcdv3, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Endpoints,
		DialTimeout: conf.ConnectionTimeout,
	})

	if err != nil {
		return
	}

	session, err := concurrency.NewSession(etcdv3)
	if err != nil {
		return
	}

	client = &EtcdClient{
		timeout: conf.RequestTimeout,
		session: session,
		etcdv3:  etcdv3,
	}
	return
}

// Get gets value from etcd by key
func (client *EtcdClient) Get(key string) (value string, ok bool, err error) {

	log := log.WithField("func", "Get").WithField("key", key).WithField("client", client)

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := client.etcdv3.Get(ctx, key)

	if err != nil {
		log.WithError(err).Error("Unable to get value by key")
		return
	}

	for _, kv := range response.Kvs {
		ok = true
		value = string(kv.Value)
		return
	}

	return
}

// GetByKeyPrefix gets all values which have the same key prefix
func (client *EtcdClient) GetByKeyPrefix(key string) (values []string, err error) {

	log := log.WithField("func", "GetByKeyPrefix").WithField("key", key).WithField("client", client)

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	keyEnd := clientv3.GetPrefixRangeEnd(key)
	response, err := client.etcdv3.Get(ctx, key, clientv3.WithRange(keyEnd))

	if err != nil {
		log.WithError(err).Error("Unable to get value by key prefix")
		return
	}

	for _, kv := range response.Kvs {
		value := string(kv.Value)
		values = append(values, value)
	}

	return
}

// Put puts key and value to etcd
func (client *EtcdClient) Put(key string, value string) (err error) {
	log := log.WithField("func", "Put").WithField("key", key).WithField("client", client)

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err = etcdv3.Put(ctx, key, value)
	if err != nil {
		log.WithError(err).Error("Unable to put value by key")
	}

	return err
}

// Delete deletes the existing key and value from etcd
func (client *EtcdClient) Delete(key string) error {
	log := log.WithField("func", "Delete").WithField("key", key).WithField("client", client)

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err := etcdv3.Delete(ctx, key)
	if err != nil {
		log.WithError(err).Error("Unable to delete value by key")
	}

	return err
}

// CompareAndSwap uses CAS operation to set a value
func (client *EtcdClient) CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error) {
	log := log.WithField("func", "CompareAndSwap").WithField("key", key).WithField("client", client)

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := etcdv3.KV.Txn(ctx).If(
		clientv3.Compare(clientv3.Value(key), "=", prevValue),
	).Then(
		clientv3.OpPut(key, newValue),
	).Commit()

	if err != nil {
		log.WithError(err).Error("Unable to compare and swap value by key")
		return false, err
	}

	return response.Succeeded, nil
}

// PutIfAbsent puts value if absent
func (client *EtcdClient) PutIfAbsent(key string, value string) (ok bool, err error) {
	log := log.WithField("func", "PutIfAbsent").WithField("key", key).WithField("client", client)

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	etcdv3 := client.etcdv3
	session, err := concurrency.NewSession(etcdv3)

	if err != nil {
		log.WithError(err).Error("Unable to create new session")
		return
	}

	// TODO: implement it using etcdv3.KV.Txn(ctx).If(...).Then(...).Commit() as in CompareAndSwap()
	mu := concurrency.NewMutex(session, key)
	err = mu.Lock(ctx)

	if err != nil {
		log.WithError(err).Error("Unable to lock mutex")
		return
	}

	defer mu.Unlock(context.Background())

	response, err := etcdv3.Get(ctx, key)

	if err != nil {
		log.WithError(err).Error("Unable to get value")
		return
	}
	if response.Count != 0 {
		return
	}

	_, err = etcdv3.Put(ctx, key, value)

	if err != nil {
		log.WithError(err).Error("Unable to put value")
		return
	}

	ok = true
	return
}

// NewMutex Create a mutex for the given key
func (client *EtcdClient) NewMutex(key string) (mutex *EtcdClientMutex, err error) {

	m := concurrency.NewMutex(client.session, key)
	mutex = &EtcdClientMutex{mutex: m}
	return
}

// Close closes etcd client
func (client *EtcdClient) Close() {
	defer client.session.Close()
	defer client.etcdv3.Close()
}
