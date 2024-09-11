package etcddb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-collections/collections/set"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/storage"
	"github.com/singnet/snet-daemon/utils"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
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

// EtcdClient struct has some useful methods to work with an etcd client
type EtcdClient struct {
	hotReaload bool
	timeout    time.Duration
	session    *concurrency.Session
	etcdv3     *clientv3.Client
}

// NewEtcdClient create new etcd storage client.
func NewEtcdClient(metaData *blockchain.OrganizationMetaData) (client *EtcdClient, err error) {
	return NewEtcdClientFromVip(config.Vip(), metaData)
}

// NewEtcdClientFromVip create new etcd storage client from viper.
func NewEtcdClientFromVip(vip *viper.Viper, metaData *blockchain.OrganizationMetaData) (client *EtcdClient, err error) {

	conf, err := GetEtcdClientConf(vip, metaData)

	if err != nil {
		return nil, err
	}

	zap.L().Info("Creating new payment storage client (etcdv3)",
		zap.String("ConnectionTimeout", conf.ConnectionTimeout.String()),
		zap.String("RequestTimeout", conf.RequestTimeout.String()),
		zap.Strings("Endpoints", conf.Endpoints))

	var etcdv3 *clientv3.Client

	if utils.CheckIfHttps(metaData.GetPaymentStorageEndPoints()) {
		tlsConfig, err := getTlsConfig()
		if err != nil {
			return nil, err
		}
		etcdv3, err = clientv3.New(clientv3.Config{
			Endpoints:   conf.Endpoints,
			DialTimeout: conf.ConnectionTimeout,
			TLS:         tlsConfig,
		})
	} else {
		// Regular http call
		etcdv3, err = clientv3.New(clientv3.Config{
			Endpoints:   conf.Endpoints,
			DialTimeout: conf.ConnectionTimeout,
		})
	}

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), conf.RequestTimeout)
	defer cancel()
	session, err := concurrency.NewSession(etcdv3, concurrency.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("can't connect to etcddb: %v", err)
	}

	client = &EtcdClient{
		hotReaload: conf.HotReload,
		timeout:    conf.RequestTimeout,
		session:    session,
		etcdv3:     etcdv3,
	}
	return
}

func Reconnect(metadata *blockchain.OrganizationMetaData) (*EtcdClient, error) {
	etcdClient, err := NewEtcdClientFromVip(config.Vip(), metadata)
	if err != nil {
		return nil, err
	}
	zap.L().Info("Successful reconnet to new etcd endpoints", zap.Strings("New endpoints", metadata.GetPaymentStorageEndPoints()))
	return etcdClient, nil
}

func getTlsConfig() (*tls.Config, error) {
	zap.L().Debug("enabling SSL support via X509 keypair")
	cert, err := tls.LoadX509KeyPair(config.GetString(config.PaymentChannelCertPath), config.GetString(config.PaymentChannelKeyPath))

	if err != nil {
		panic("unable to load specific SSL X509 keypair for etcd")
	}
	caCert, err := os.ReadFile(config.GetString(config.PaymentChannelCaPath))
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	return tlsConfig, nil
}

// Get gets value from etcd by key
func (client *EtcdClient) Get(key string) (value string, ok bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := client.etcdv3.Get(ctx, key)

	if err != nil {
		zap.L().Error("Unable to get value by key",
			zap.Error(err),
			zap.String("func", "Get"),
			zap.String("key", key),
			zap.Any("client", client))
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

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	keyEnd := clientv3.GetPrefixRangeEnd(key)
	response, err := client.etcdv3.Get(ctx, key, clientv3.WithRange(keyEnd))

	if err != nil {
		zap.L().Error("Unable to get value by key prefix",
			zap.Error(err),
			zap.String("func", "Get"),
			zap.String("key", key),
			zap.Any("client", client))
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

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err = etcdv3.Put(ctx, key, value)
	if err != nil {
		zap.L().Error("Unable to put value by key",
			zap.Error(err),
			zap.String("func", "Put"),
			zap.String("key", key),
			zap.Any("client", client))
	}

	return err
}

// Delete deletes the existing key and value from etcd
func (client *EtcdClient) Delete(key string) error {

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err := etcdv3.Delete(ctx, key)
	if err != nil {
		zap.L().Error("Unable to delete value by key",
			zap.Error(err),
			zap.String("func", "Delete"),
			zap.String("key", key),
			zap.Any("client", client))
	}

	return err
}

// EtcdKeyValue contains key and value
type EtcdKeyValue struct {
	key   string
	value string
}

func NewEtcdKeyValue(key string, value string) EtcdKeyValue {
	return EtcdKeyValue{key: key, value: value}
}

// CompareAndSwap uses CAS operation to set a value
func (client *EtcdClient) CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error) {
	return client.ExecuteTransaction(storage.CASRequest{
		RetryTillSuccessOrError: false,
		ConditionKeys:           []string{key},
		Update: func(oldValues []storage.KeyValueData) (update []storage.KeyValueData, ok bool, err error) {
			if oldValues[0].Present && strings.Compare(oldValues[0].Value, prevValue) == 0 {
				return []storage.KeyValueData{{
					Key:   key,
					Value: newValue,
				}}, true, nil
			} else {
				return nil, false, nil
			}
		},
	})
}

// Transaction uses CAS operation to compare and set multiple key values
func (client *EtcdClient) Transaction(compare []EtcdKeyValue, swap []EtcdKeyValue) (ok bool, err error) {

	etcdv3 := client.etcdv3
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	cmps := make([]clientv3.Cmp, len(compare))

	for index, cmp := range compare {
		cmps[index] = clientv3.Compare(clientv3.Value(cmp.key), "=", cmp.value)
	}

	ops := make([]clientv3.Op, len(swap))
	for index, op := range swap {
		ops[index] = clientv3.OpPut(op.key, op.value)
	}

	response, err := etcdv3.KV.Txn(ctx).If(cmps...).Then(ops...).Commit()

	if err != nil {
		keys := []string{}
		for _, keyValue := range compare {
			keys = append(keys, keyValue.key)
		}
		zap.L().Error("Unable to compare and swap value by keys",
			zap.Error(err),
			zap.String("keys", strings.Join(keys, ", ")),
			zap.String("func", "CompareAndSwap"),
			zap.Any("client", client))
		return false, err
	}

	return response.Succeeded, nil
}

// PutIfAbsent puts value if absent
func (client *EtcdClient) PutIfAbsent(key string, value string) (ok bool, err error) {
	return client.ExecuteTransaction(storage.CASRequest{
		RetryTillSuccessOrError: false,
		ConditionKeys:           []string{key},
		Update: func(oldValues []storage.KeyValueData) (update []storage.KeyValueData, ok bool, err error) {
			if oldValues[0].Present {
				return nil, false, nil
			}
			return []storage.KeyValueData{{
				Key:   key,
				Value: value,
			}}, true, nil
		},
	})
}

// NewMutex Create a mutex for the given key
func (client *EtcdClient) NewMutex(key string) (mutex *EtcdClientMutex, err error) {

	m := concurrency.NewMutex(client.session, key)
	mutex = &EtcdClientMutex{mutex: m}
	return
}

func (client *EtcdClient) ExecuteTransaction(request storage.CASRequest) (ok bool, err error) {

	transaction, err := client.StartTransaction(request.ConditionKeys)
	if err != nil {
		return false, err
	}
	//We should also have a configuration on how many times you try this ( say 100 times )
	for {
		oldValues, err := transaction.GetConditionValues()
		if err != nil {
			return false, err
		}
		newValues, ok, err := request.Update(oldValues)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		if ok, err = client.CompleteTransaction(transaction, newValues); err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if !request.RetryTillSuccessOrError {
			return false, nil
		}
	}
}

// If there are no Old values in the transaction, to compare, then this method
// can be used to write in the new values , if the key does not exist then put it in a transaction
func (client *EtcdClient) CompleteTransaction(_transaction storage.Transaction, update []storage.KeyValueData) (
	ok bool, err error) {

	var transaction *etcdTransaction = _transaction.(*etcdTransaction)

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()
	defer ctx.Done()
	startime := time.Now()
	txn := client.etcdv3.KV.Txn(ctx)
	var txnResp *clientv3.TxnResponse
	conditionKeys := transaction.ConditionKeys

	ifCompares := make([]clientv3.Cmp, len(transaction.ConditionValues))
	for i, cmp := range transaction.ConditionValues {
		if cmp.Present {
			ifCompares[i] = clientv3.Compare(clientv3.ModRevision(cmp.Key), "=", cmp.Version)
		} else {
			ifCompares[i] = clientv3.Compare(clientv3.CreateRevision(cmp.Key), "=", 0)
		}
	}

	thenOps := make([]clientv3.Op, len(update))
	for index, op := range update {
		// TODO: add OpDelete if op.Present is false
		thenOps[index] = clientv3.OpPut(op.Key, op.Value)
	}

	elseOps := make([]clientv3.Op, len(conditionKeys))
	for index, key := range conditionKeys {
		elseOps[index] = clientv3.OpGet(key)
	}
	txnResp, err = txn.If(ifCompares...).Then(thenOps...).Else(elseOps...).Commit()

	endtime := time.Now()

	zap.L().Debug("etcd transaction time", zap.Any("time", endtime.Sub(startime)))
	if err != nil {
		return false, err
	}

	if txnResp == nil {
		return false, fmt.Errorf("transaction response is nil")
	}
	if txnResp.Succeeded {
		return true, nil
	}

	var latestValues []keyValueVersion
	if latestValues, err = client.checkTxnResponse(conditionKeys, txnResp); err != nil {
		return false, err
	}

	transaction.ConditionValues = latestValues
	//succeeded is set to true if the compare evaluated to true or false otherwise.
	return txnResp.Succeeded, nil
}

func (client *EtcdClient) checkTxnResponse(keys []string, txnResp *clientv3.TxnResponse) (latestStateArray []keyValueVersion, err error) {
	keySet := set.New()
	for _, key := range keys {
		keySet.Insert(key)
	}
	// FIXME: allocate len(keys) array
	latestStateArray = make([]keyValueVersion, 0)
	for _, response := range txnResp.Responses {
		txnGetValue := (*clientv3.GetResponse)(response.GetResponseRange())
		latestValues, err := client.getState(keySet, txnGetValue)
		if err != nil {
			return nil, err
		}
		latestStateArray = append(latestStateArray, latestValues...)
	}
	keySet.Do(func(elem any) {
		latestStateArray = append(latestStateArray, keyValueVersion{
			Key:     elem.(string),
			Present: false,
		})
	})
	return latestStateArray, nil

}

func (client *EtcdClient) getState(keySet *set.Set, getResp *clientv3.GetResponse) (latestStateArray []keyValueVersion, err error) {
	latestStateArray = make([]keyValueVersion, len(getResp.Kvs))
	for i, eachResponse := range getResp.Kvs {
		state := keyValueVersion{
			Present: true,
			Version: eachResponse.ModRevision,
			Value:   string(eachResponse.Value),
			Key:     string(eachResponse.Key),
		}
		keySet.Remove(state.Key)
		latestStateArray[i] = state
	}
	return latestStateArray, nil
}

// Close closes etcd client
func (client *EtcdClient) Close() {
	defer client.session.Close()
	defer client.etcdv3.Close()
}

func (client *EtcdClient) StartTransaction(keys []string) (_transaction storage.Transaction, err error) {
	transaction := &etcdTransaction{
		ConditionKeys: keys,
	}
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	ops := make([]clientv3.Op, len(keys))
	for i, key := range keys {
		ops[i] = clientv3.OpGet(key)
	}

	txn := client.etcdv3.KV.Txn(ctx)
	txn.Then(ops...)
	txnResp, err := txn.Commit()

	if err != nil {
		zap.L().Error("error in getting values", zap.Error(err))
		return nil, err
	}
	if txnResp != nil {
		if latestValues, err := client.checkTxnResponse(keys, txnResp); err != nil {
			return nil, err
		} else {
			transaction.ConditionValues = latestValues
		}
	}

	return transaction, nil
}

func (client *EtcdClient) IsHotReloadEnabled() bool {
	return client.hotReaload
}

type keyValueVersion struct {
	Key     string
	Value   string
	Present bool
	Version int64
}

type etcdTransaction struct {
	ConditionValues []keyValueVersion
	ConditionKeys   []string
}

func (transaction *etcdTransaction) GetConditionValues() ([]storage.KeyValueData, error) {
	values := make([]storage.KeyValueData, len(transaction.ConditionValues))
	for i, value := range transaction.ConditionValues {
		values[i] = storage.KeyValueData{
			Key:     value.Key,
			Value:   value.Value,
			Present: value.Present,
		}
	}
	return values, nil
}
