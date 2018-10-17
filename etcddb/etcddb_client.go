package etcddb

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.etcd.io/etcd/clientv3"
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
// It uses default connection and request timeouts in case
// PAYMENT_CHANNEL_STORAGE_CLIENT struct is not set
func NewEtcdClient(vip *viper.Viper) (client *EtcdClient, err error) {

	cluster := GetPaymentChannelCluster(vip)
	endpoints, err := getPaymentChannelEndpoints(cluster)

	if err != nil {
		return
	}

	conf, err := GetPaymentChannelStorageClientConf(vip)

	if err != nil {
		return
	}

	if conf == nil {
		conf = &PaymentChannelStorageClientConf{
			ConnectionTimeout: DefaultConnectionTimeout,
			RequestTimeout:    DefaultRequestTimeout,
		}
	}

	connectionTimeout := time.Duration(conf.ConnectionTimeout) * time.Millisecond

	etcdv3, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
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
func (client *EtcdClient) Get(key []byte) ([]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := client.etcdv3.Get(ctx, byteArraytoString(key))

	if err != nil {
		return nil, err
	}

	for _, kv := range response.Kvs {
		return kv.Value, nil
	}
	return nil, nil
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

func getPaymentChannelEndpoints(cluster string) (endpoints []string, err error) {

	for _, nameAndEndpoint := range strings.Split(cluster, ",") {
		nameAndEndpoint = strings.TrimSpace(nameAndEndpoint)
		index := strings.Index(nameAndEndpoint, "=")
		if index <= 0 || index >= len(nameAndEndpoint)-1 {
			err = errors.New("cluster string does not have format name=host:port[,name=host:port]+ " + cluster)
			return
		}
		endpoint := nameAndEndpoint[index+1 : len(nameAndEndpoint)]
		endpoints = append(endpoints, endpoint)
	}

	return
}
