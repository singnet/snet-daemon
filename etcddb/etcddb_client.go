package etcddb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

const etcdTTL = 10

// EtcdClientMutex mutex struct for etcd client
type EtcdClientMutex struct {
	mutex *concurrency.Mutex
}

// Lock etcd key
func (mutex *EtcdClientMutex) Lock(ctx context.Context) (err error) {
	return mutex.mutex.Lock(ctx)
}

// Unlock unlock etcd key
func (mutex *EtcdClientMutex) Unlock(ctx context.Context) (err error) {
	return mutex.mutex.Unlock(ctx)
}

// EtcdClient struct has some useful methods to work with an etcd client
type EtcdClient struct {
	hotReload bool
	timeout   time.Duration
	session   *concurrency.Session
	etcd      *clientv3.Client
}

var _ storage.AtomicStorage = (*EtcdClient)(nil)

// NewEtcdClient create new etcd storage client.
func NewEtcdClient(metaData *blockchain.OrganizationMetaData) (client *EtcdClient, err error) {
	return NewEtcdClientFromVip(config.Vip(), metaData)
}

// NewEtcdClientFromVip creates a new EtcdClient using settings from the given Viper instance.
// It performs a bounded-time health probe (Maintenance.Status) to fail fast when endpoints
// are unreachable or the cluster is unhealthy, and then creates a long-lived etcd concurrency
// session without attaching a per-call deadline (so shutdown can revoke the lease cleanly).
//
// Behavior:
//   - Respects connection and request timeouts from config for dialing and the health probe.
//   - If HTTPS endpoints are provided, a TLS config is built via getTLSConfig().
//   - On any initialization error, the underlying client is closed before returning.
//
// Returns:
//   - (*EtcdClient, nil) on success;
//   - (nil, error) if dialing, health probe, or session creation fails.
//
// Note: The session is created without a custom context to avoid canceling it prematurely
// during shutdown. Use EtcdClient.Close() to gracefully revoke the lease and close resources.
func NewEtcdClientFromVip(vip *viper.Viper, metaData *blockchain.OrganizationMetaData) (client *EtcdClient, err error) {

	conf, err := GetEtcdClientConf(vip, metaData)

	if err != nil {
		return nil, err
	}

	zap.L().Info("Creating new payment storage client (etcd)",
		zap.String("ConnectionTimeout", conf.ConnectionTimeout.String()),
		zap.String("RequestTimeout", conf.RequestTimeout.String()),
		zap.Strings("Endpoints", conf.Endpoints))

	var etcdv3 *clientv3.Client

	cfg := clientv3.Config{
		Endpoints:            conf.Endpoints,
		DialTimeout:          conf.ConnectionTimeout,
		DialKeepAliveTime:    10 * time.Second,
		DialKeepAliveTimeout: 3 * time.Second,
	}

	if utils.CheckIfHttps(conf.Endpoints) {
		var tlsConfig *tls.Config
		tlsConfig, err = getTLSConfig()
		if err != nil {
			return nil, err
		}
		cfg.TLS = tlsConfig
	}

	etcdv3, err = clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	// Fast-fail health probe with a bounded request timeout so initialization never hangs.
	probeCtx, cancel := context.WithTimeout(context.Background(), conf.RequestTimeout)
	defer cancel()

	// Probe the first endpoint (you may loop all endpoints if desired).
	if _, err := etcdv3.Maintenance.Status(probeCtx, conf.Endpoints[0]); err != nil {
		_ = etcdv3.Close()
		return nil, fmt.Errorf("etcd not healthy: %w", err)
	}

	// Create a long-lived session (uses context.Background under the hood).
	// This avoids tying the session lifecycle to a short timeout context,
	// which would cause LeaseRevoke to see "context canceled" during shutdown.
	session, err := concurrency.NewSession(etcdv3, concurrency.WithTTL(etcdTTL))
	if err != nil {
		_ = etcdv3.Close()
		return nil, fmt.Errorf("can't create etcd session: %w", err)
	}

	zap.L().Debug("[etcd] session created")

	// Log when the session is closed (e.g., due to connection loss).
	go func() {
		<-session.Done()
		zap.L().Debug("[etcd] session closed")
	}()

	client = &EtcdClient{
		hotReload: conf.HotReload,
		timeout:   conf.RequestTimeout,
		session:   session,
		etcd:      etcdv3,
	}
	return
}

func Reconnect(metadata *blockchain.OrganizationMetaData) (*EtcdClient, error) {
	etcdClient, err := NewEtcdClientFromVip(config.Vip(), metadata)
	if err != nil {
		return nil, err
	}
	zap.L().Info("[etcd] Successful reconnect to new etcd endpoints", zap.Strings("New endpoints", metadata.GetPaymentStorageEndPoints()))
	return etcdClient, nil
}

func getTLSConfig() (*tls.Config, error) {

	certPath := config.GetString(config.PaymentChannelCertPath)
	keyPath := config.GetString(config.PaymentChannelKeyPath)
	caPath := config.GetString(config.PaymentChannelCaPath)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		zap.L().Error("[etcd] unable to load SSL X509 keypair",
			zap.String("certPath", certPath),
			zap.String("keyPath", keyPath),
			zap.Error(err),
		)
		return nil, fmt.Errorf("load x509 keypair: %w", err)
	}

	if len(cert.Certificate) > 0 {
		if parsed, _ := x509.ParseCertificate(cert.Certificate[0]); parsed != nil {
			zap.L().Debug("[etcd] client cert EKU", zap.Any("eku", parsed.ExtKeyUsage), zap.String("subject", parsed.Subject.String()))
		}
	}

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	zap.L().Debug("[etcd] enabling SSL support via X509 keypair")
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	return tlsConfig, nil
}

// Get - get value from etcd by key
func (client *EtcdClient) Get(key string) (value string, ok bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	response, err := client.etcd.Get(ctx, key)

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

// GetByKeyPrefix gets all values that have the same key prefix
func (client *EtcdClient) GetByKeyPrefix(key string) (values []string, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	keyEnd := clientv3.GetPrefixRangeEnd(key)
	response, err := client.etcd.Get(ctx, key, clientv3.WithRange(keyEnd))

	if err != nil {
		zap.L().Error("Unable to get value by key prefix",
			zap.Error(err),
			zap.String("func", "GetByKeyPrefix"),
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

	etcdv3 := client.etcd
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

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	_, err := client.etcd.Delete(ctx, key)
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
			if oldValues[0].Present && oldValues[0].Value == prevValue {
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

	response, err := client.etcd.KV.Txn(ctx).If(cmps...).Then(ops...).Commit()

	if err != nil {
		keys := make([]string, 0, len(compare))
		for _, keyValue := range compare {
			keys = append(keys, keyValue.key)
		}
		zap.L().Error("Unable to execute transaction",
			zap.Error(err),
			zap.String("keys", strings.Join(keys, ", ")),
			zap.String("func", "Transaction"),
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

// CompleteTransaction atomically applies updates if the etcd state still matches
// the snapshot captured by StartTransaction. The _transaction must be an
// *etcdTransaction; otherwise an error is returned.
//
// Compare rules per condition key:
//   - Present == true  → ModRevision(key) == Version
//   - Present == false → CreateRevision(key) == 0 (key must not exist)
//
// On success, applies updates with OpPut(key, value) in a single txn.
// Note: delete on update is not supported yet; `Present` only affects the compare stage.
//
// On compare failure, returns ok == false and refreshes ConditionValues for retry.
// Returns ok == true on success; err != nil on client/commit errors. Respects the request timeout
// and logs txn latency at DEBUG level.
func (client *EtcdClient) CompleteTransaction(_transaction storage.Transaction, update []storage.KeyValueData) (
	ok bool, err error) {

	transaction, okType := _transaction.(*etcdTransaction)
	if !okType {
		return false, fmt.Errorf("unexpected transaction type: %T", _transaction)
	}

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()
	startTime := time.Now()
	txn := client.etcd.KV.Txn(ctx)
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
	for i, op := range update {
		// NOTE: `Present` is only used for compare stage; updates always PUT for now.
		// TODO: delete operation is not supported yet.
		//if !op.Present {
		//	thenOps = append(thenOps, clientv3.OpDelete(op.Key))
		//	continue
		//}
		thenOps[i] = clientv3.OpPut(op.Key, op.Value)
	}

	elseOps := make([]clientv3.Op, len(conditionKeys))
	for index, key := range conditionKeys {
		elseOps[index] = clientv3.OpGet(key)
	}
	txnResp, err = txn.If(ifCompares...).Then(thenOps...).Else(elseOps...).Commit()

	zap.L().Debug("etcd transaction time", zap.Duration("latency", time.Since(startTime)))

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

func (client *EtcdClient) checkTxnResponse(keys []string, txnResp *clientv3.TxnResponse) ([]keyValueVersion, error) {
	keySet := make(map[string]struct{})
	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	latestStateArray := make([]keyValueVersion, 0, len(keys))

	for _, response := range txnResp.Responses {
		txnGetValue := (*clientv3.GetResponse)(response.GetResponseRange())
		latestValues, err := client.getState(keySet, txnGetValue)
		if err != nil {
			return nil, err
		}
		latestStateArray = append(latestStateArray, latestValues...)
	}

	for key := range keySet {
		latestStateArray = append(latestStateArray, keyValueVersion{
			Key:     key,
			Present: false,
		})
	}

	return latestStateArray, nil
}

func (client *EtcdClient) getState(keySet map[string]struct{}, getResp *clientv3.GetResponse) ([]keyValueVersion, error) {
	latestStateArray := make([]keyValueVersion, len(getResp.Kvs))
	for i, eachResponse := range getResp.Kvs {
		key := string(eachResponse.Key)
		state := keyValueVersion{
			Present: true,
			Version: eachResponse.ModRevision,
			Value:   string(eachResponse.Value),
			Key:     key,
		}
		delete(keySet, key)
		latestStateArray[i] = state
	}
	return latestStateArray, nil
}

func (client *EtcdClient) Close() {
	if client.session != nil {
		if err := client.session.Close(); err != nil {
			zap.L().Error("close session failed", zap.Error(err))
		}
	}
	if client.etcd != nil {
		if err := client.etcd.Close(); err != nil {
			zap.L().Error("close etcd client failed", zap.Error(err))
		}
	}
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

	txn := client.etcd.KV.Txn(ctx)
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
	return client.hotReload
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

var _ storage.Transaction = (*etcdTransaction)(nil)

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
