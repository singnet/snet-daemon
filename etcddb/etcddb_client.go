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

	if err != nil {
		return nil, err
	}

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

	return client.Transaction(
		[]EtcdKeyValue{EtcdKeyValue{key: key, value: prevValue}},
		[]EtcdKeyValue{EtcdKeyValue{key: key, value: newValue}},
	)
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

//we can make this exposed , if there are no Old values, to compare, then this method
//can be used to write in the new values , if the key does not exist then put it in a transaction

func (client *EtcdClient) etcdCas(request *escrow.CASRequest) (response *escrow.CASResponse, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout*time.Second*100)
	defer cancel()
	defer ctx.Done()

	txn := client.etcdv3.KV.Txn(ctx)
	var txnResp *clientv3.TxnResponse
	if txn, err = client.buildIf(txn, request); err != nil {
		return
	}
	if txn, err = client.buildThenOperations(txn, request); err != nil {
		return
	}
	if txn, err = client.buildElseOperations(txn, request); err != nil {
		return
	}
	txnResp, err = txn.Commit()

	if err != nil {
		return response, err
	}

	if txnResp != nil {
		response = &escrow.CASResponse{}
		//succeeded is set to true if the compare evaluated to true or false otherwise.
		response.Succeeded = txnResp.Succeeded
		if latestValues, err := client.checkTxnResponse(txnResp); err != nil {
			return nil, err
		} else {
			response.LatestData = latestValues
		}
	}
	return response, err
}

func (client *EtcdClient) buildIf(txn clientv3.Txn, request *escrow.CASRequest) (clientv3.Txn, error) {

	cmps := make([]clientv3.Cmp, len(request.OldKeyValues))

	for i, cmp := range request.OldKeyValues {
		operator := fmt.Sprint(cmp.Compare.Operator)
		switch cmp.Compare.CompareOn {
		case escrow.VALUE:
			//this is a Lexical comparision , please note "10" < "4" in Lexical comparision
			//you need "10" < "04" (padded zeros to compare the value accurately)
			// - Not Ideal if you have numbers to check
			cmps[i] = clientv3.Compare(clientv3.Value(cmp.Key), operator, cmp.Value)
		case escrow.MODIFIED_VERSION:
			cmps[i] = clientv3.Compare(clientv3.ModRevision(cmp.Key), operator, cmp.Version)
		default:
			return nil, fmt.Errorf("Unknown Type of Comparision op")
		}
	}
	return txn.If(cmps...), nil
}

func (client *EtcdClient) buildThenOperations(txn clientv3.Txn, request *escrow.CASRequest) (clientv3.Txn, error) {
	ops := make([]clientv3.Op, len(request.NewKeyValues))
	for index, op := range request.NewKeyValues {
		if op == nil {
			continue
		}
		ops[index] = clientv3.OpPut(op.Key, op.Value)
	}
	return txn.Then(ops...), nil
}

func (client *EtcdClient) buildElseOperations(txn clientv3.Txn, request *escrow.CASRequest) (clientv3.Txn, error) {
	return txn.Else(clientv3.OpGet(request.KeyPrefix, clientv3.WithPrefix())), nil
}

func (client *EtcdClient) checkTxnResponse(txnResp *clientv3.TxnResponse) (latestStateArray []*escrow.KeyValueData, err error) {

	if !txnResp.Succeeded {
		latestStateArray = make([]*escrow.KeyValueData, 0)
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
	return nil, nil
}

func (client *EtcdClient) getState(getResp *clientv3.GetResponse) (latestStateArray []*escrow.KeyValueData, err error) {

	if len(getResp.Kvs) == 0 {
		return nil, nil
	} else {
		latestStateArray = make([]*escrow.KeyValueData, len(getResp.Kvs))
		for _, eachResponse := range getResp.Kvs {
			state := &escrow.KeyValueData{}
			state.Version = eachResponse.ModRevision
			state.Value = string(eachResponse.Value)
			state.Key = string(eachResponse.Key)
			latestStateArray = append(latestStateArray, state)
		}
	}
	return latestStateArray, nil
}

// Close closes etcd client
func (client *EtcdClient) Close() {
	defer client.session.Close()
	defer client.etcdv3.Close()
}

func (client *EtcdClient) CAS(request *escrow.CASRequest) (*escrow.CASResponse, error) {
	for {
		transaction := client.
			casTransaction(request.KeyPrefix, request.AdditionalParameters).
			If(request.Condition).
			Then(request.Action)
		if transaction.err != nil {
			return nil, transaction.err
		}
		if !transaction.transactionResult && request.RetryTillSuccessOrError {
			continue
		}
		return &escrow.CASResponse{Succeeded: transaction.transactionResult,
			LatestData: transaction.latestValuesRead}, transaction.err
	}
}

func (client *EtcdClient) casTransaction(key string, params interface{}) (transaction *casTransaction) {
	transaction = &casTransaction{client: client, additionalParams: params}
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	keyEnd := clientv3.GetPrefixRangeEnd(key)
	txnResp, err := client.etcdv3.Get(ctx, key, clientv3.WithRange(keyEnd))

	if err != nil {
		log.WithError(err).Error("error in getting value by key prefix")
		return
	}
	if txnResp != nil {

		if latestValues, err := client.getState(txnResp); err != nil {
			transaction.err = err
			return transaction
		} else {
			transaction.latestValuesRead = latestValues
		}
	}

	return transaction
}

type casTransaction struct {
	client            *EtcdClient
	key               string
	transactionResult bool
	err               error
	latestValuesRead  []*escrow.KeyValueData
	newValuesWrite    []*escrow.KeyValueData
	businessData      interface{}
	additionalParams  interface{}
}

func (transaction *casTransaction) If(condition escrow.ConditionFunc) *casTransaction {
	//Do the next steps ONLY if there is no error
	if transaction.err != nil {
		return transaction
	}
	if newValuesDetermined, err := condition(transaction.latestValuesRead, transaction.additionalParams); err != nil {
		transaction.err = err
		return transaction
	} else {
		transaction.businessData = newValuesDetermined
	}

	return transaction
}
func (transaction *casTransaction) Then(action escrow.ActionFunc) *casTransaction {
	if transaction.err != nil {
		return transaction
	}

	oldKeyValues, newKeyValues, err := action(transaction.businessData)
	if err != nil {
		transaction.err = err
		return transaction
	}
	request := &escrow.CASRequest{NewKeyValues: newKeyValues, OldKeyValues: oldKeyValues}
	if response, err := transaction.client.etcdCas(request); err != nil {
		transaction.err = err
		return transaction
	} else {
		transaction.transactionResult = response.Succeeded
	}
	return transaction

}

/*todo for future use !
func (transaction *casTransaction) Else(action func()) *casTransaction {
	return transaction
}*/
