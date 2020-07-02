package etcddb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/escrow"
	"io/ioutil"
	"strings"
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
func NewEtcdClient(metaData *blockchain.OrganizationMetaData) (client *EtcdClient, err error) {
	return NewEtcdClientFromVip(config.Vip(), metaData)
}

// NewEtcdClientFromVip create new etcd storage client from viper.
func NewEtcdClientFromVip(vip *viper.Viper, metaData *blockchain.OrganizationMetaData) (client *EtcdClient, err error) {

	conf, err := GetEtcdClientConf(vip, metaData)

	if err != nil {
		return nil, err
	}

	log.WithField("PaymentChannelStorageClient", fmt.Sprintf("%+v", conf)).Info()

	var etcdv3 *clientv3.Client

	if checkIfHttps(metaData.GetPaymentStorageEndPoints()) {
		if tlsConfig, err := getTlsConfig(); err == nil {
			etcdv3, err = clientv3.New(clientv3.Config{
				Endpoints:   metaData.GetPaymentStorageEndPoints(),
				DialTimeout: conf.ConnectionTimeout,
				TLS:         tlsConfig,
			})
		} else {
			return nil, err
		}

	} else {
		//Regular http call
		etcdv3, err = clientv3.New(clientv3.Config{
			Endpoints:   metaData.GetPaymentStorageEndPoints(),
			DialTimeout: conf.ConnectionTimeout,
		})
		if err != nil {
			return nil, err
		}
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
func getTlsConfig() (*tls.Config, error) {

	log.Debug("enabling SSL support via X509 keypair")
	cert, err := tls.LoadX509KeyPair(config.GetString(config.PaymentChannelCertPath), config.GetString(config.PaymentChannelKeyPath))

	if err != nil {
		panic("unable to load specific SSL X509 keypair for etcd")
	}
	caCert, err := ioutil.ReadFile(config.GetString(config.PaymentChannelCaPath))
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

func checkIfHttps(endpoints []string) bool {
	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "https") {
			return true
		}
	}
	return false
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

// EtcdKeyValue contains key and value
type EtcdKeyValue struct {
	key   string
	value string
}

// CompareAndSwap uses CAS operation to set a value
func (client *EtcdClient) CompareAndSwap(key string, prevValue string, newValue string) (ok bool, err error) {

	transaction, err := client.StartTransaction([]string{key})
	if err != nil {
		return false, err
	}
	update := make([]escrow.KeyValueData, 0)
	values, err := transaction.GetConditionValues()
	if err != nil {
		return false, err
	}
	if strings.Compare(values[0].Value, prevValue) == 0 {
		update = append(update, escrow.KeyValueData{Key: key, Value: newValue})
		return client.CompleteTransaction(transaction, update)
	}
	return false, nil
}

// Transaction uses CAS operation to compare and set multiple key values
func (client *EtcdClient) Transaction(compare []EtcdKeyValue, swap []EtcdKeyValue) (ok bool, err error) {

	log := log.WithField("func", "CompareAndSwap").WithField("client", client)

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
		log = log.WithField("keys", strings.Join(keys, ", "))
		log.WithError(err).Error("Unable to compare and swap value by keys")
		return false, err
	}

	return response.Succeeded, nil
}

// PutIfAbsent puts value if absent
func (client *EtcdClient) PutIfAbsent(key string, value string) (ok bool, err error) {
	log := log.WithField("func", "PutIfAbsent").WithField("key", key).WithField("client", client)

	transaction, err := client.StartTransaction([]string{key})
	if err != nil {
		log.WithError(err).Error("Error in PutIfAbsent while trying to retrieve key")
		return false, err
	}
	values, err := transaction.GetConditionValues()
	if err != nil {
		return false, err
	}
	if len(values) == 0 || !values[0].Present {
		update := make([]escrow.KeyValueData, 0)
		update = append(update, escrow.KeyValueData{Key: key, Value: value})
		return client.CompleteTransaction(transaction, update)
	}
	return false, nil
}

// NewMutex Create a mutex for the given key
func (client *EtcdClient) NewMutex(key string) (mutex *EtcdClientMutex, err error) {

	m := concurrency.NewMutex(client.session, key)
	mutex = &EtcdClientMutex{mutex: m}
	return
}

func (client *EtcdClient) ExecuteTransaction(request escrow.CASRequest) (ok bool, err error) {

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
		newValues, err := request.Update(oldValues)
		if err != nil {
			return false, err
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

//If there are no Old values in the transaction, to compare, then this method
//can be used to write in the new values , if the key does not exist then put it in a transaction

func (client *EtcdClient) CompleteTransaction(_transaction escrow.Transaction, update []escrow.KeyValueData) (
	ok bool, err error) {

	var transaction *etcdTransaction = _transaction.(*etcdTransaction)

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()
	defer ctx.Done()

	txn := client.etcdv3.KV.Txn(ctx)
	var txnResp *clientv3.TxnResponse
	if txn, err = client.buildIf(txn, transaction); err != nil {
		return false, err
	}
	if txn, err = client.buildThenOperations(txn, update); err != nil {
		return false, err
	}
	if txn, err = client.buildElseOperations(txn, GetKeysFromKeyValueData(update)); err != nil {
		return false, err
	}
	txnResp, err = txn.Commit()

	if err != nil {
		return false, err
	}

	if txnResp == nil {
		return false, fmt.Errorf("transaction response is nil")
	}
	if txnResp.Succeeded {
		return true, nil
	}

	var latestValues []*keyValueVersion
	if latestValues, err = client.checkTxnResponse(txnResp); err != nil {
		return false, err
	}

	transaction.ConditionValues = latestValues
	//succeeded is set to true if the compare evaluated to true or false otherwise.
	return txnResp.Succeeded, nil
}

func GetKeysFromKeyValueData(update []escrow.KeyValueData) []string {
	keys := make([]string, len(update))
	for i, key := range update {
		keys[i] = key.Key
	}
	return keys
}

func (client *EtcdClient) buildIf(txn clientv3.Txn, transaction *etcdTransaction) (clientv3.Txn, error) {
	if len(transaction.ConditionValues) == 0 {
		return txn.If(client.alwaysTrueCompare()), nil
	}
	cmps := make([]clientv3.Cmp, len(transaction.ConditionValues))

	for i, cmp := range transaction.ConditionValues {
		cmps[i] = clientv3.Compare(clientv3.ModRevision(cmp.Key), "=", cmp.Version)
	}
	return txn.If(cmps...), nil
}

func (client *EtcdClient) alwaysTrueCompare() clientv3.Cmp {
	return clientv3.Compare(clientv3.ModRevision("dummyKey"), "=", 0)
}

func (client *EtcdClient) buildThenOperations(txn clientv3.Txn, update []escrow.KeyValueData) (clientv3.Txn, error) {
	ops := make([]clientv3.Op, len(update))
	for index, op := range update {
		ops[index] = clientv3.OpPut(op.Key, op.Value)
	}
	return txn.Then(ops...), nil
}

func (client *EtcdClient) buildElseOperations(txn clientv3.Txn, conditionKeys []string) (clientv3.Txn, error) {
	ops := make([]clientv3.Op, len(conditionKeys))
	for index, key := range conditionKeys {
		ops[index] = clientv3.OpGet(key)
	}
	return txn.Else(ops...), nil
}

func (client *EtcdClient) checkTxnResponse(txnResp *clientv3.TxnResponse) (latestStateArray []*keyValueVersion, err error) {

	//if !txnResp.Succeeded {
	latestStateArray = make([]*keyValueVersion, 0)
	for _, response := range txnResp.Responses {
		txnGetValue := (*clientv3.GetResponse)(response.GetResponseRange())
		latestValues, err := client.getState(txnGetValue)
		if err != nil {
			return nil, err
		}
		latestStateArray = append(latestStateArray, latestValues...)
	}
	return latestStateArray, nil

}

func (client *EtcdClient) getState(getResp *clientv3.GetResponse) (latestStateArray []*keyValueVersion, err error) {

	if len(getResp.Kvs) == 0 {
		return nil, nil
	} else {
		latestStateArray = make([]*keyValueVersion, len(getResp.Kvs))
		for i, eachResponse := range getResp.Kvs {
			state := &keyValueVersion{}
			state.Version = eachResponse.ModRevision
			state.Value = string(eachResponse.Value)
			state.Key = string(eachResponse.Key)
			latestStateArray[i] = state
		}
	}
	return latestStateArray, nil
}

// Close closes etcd client
func (client *EtcdClient) Close() {
	defer client.session.Close()
	defer client.etcdv3.Close()
}

func (client *EtcdClient) StartTransaction(keys []string) (_transaction escrow.Transaction, err error) {
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
	//Goal is to read all the key values in one Shot !
	//todo is there a better way to read all values in one transaction
	txn.If(client.alwaysTrueCompare()).Then(ops...)
	txnResp, err := txn.Commit()

	if err != nil {
		log.WithError(err).Error("error in getting value by key prefix")
		return nil, err
	}
	if txnResp != nil {

		if latestValues, err := client.checkTxnResponse(txnResp); err != nil {
			return nil, err
		} else {
			transaction.ConditionValues = latestValues
		}
	}

	return transaction, nil
}

type keyValueVersion struct {
	Key     string
	Version int64
	Value   string
}

type etcdTransaction struct {
	ConditionValues []*keyValueVersion
	ConditionKeys   []string
}

func (transaction *etcdTransaction) GetConditionValues() ([]escrow.KeyValueData, error) {
	values := make([]escrow.KeyValueData, len(transaction.ConditionValues))
	for i, value := range transaction.ConditionValues {
		values[i] = escrow.KeyValueData{
			Key:     value.Key,
			Value:   value.Value,
			Present: true,
		}
	}
	return values, nil
}
