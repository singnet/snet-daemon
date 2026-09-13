package etcddb

import (
	"context"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *EtcdTestSuite) TestExpiredRequestsDoNotWrite() {
	t := suite.T()
	key := t.Name()
	require.NoError(t, suite.client.Put(key, "original"))
	txn, err := suite.client.StartTransaction([]string{key})
	require.NoError(t, err)
	// Share the live connection, but expire requests before they can reach etcd.
	client := *suite.client
	client.timeout = -time.Second

	value, present, err := client.Get(key)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, present)
	require.Empty(t, value)
	values, err := client.GetByKeyPrefix(key)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, values)
	require.ErrorIs(t, client.Put(key, "unexpected"), context.DeadlineExceeded)
	require.ErrorIs(t, client.Delete(key), context.DeadlineExceeded)
	ok, err := client.Transaction([]EtcdKeyValue{NewEtcdKeyValue(key, "original")},
		[]EtcdKeyValue{NewEtcdKeyValue(key, "unexpected")})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, ok)
	failedTxn, err := client.StartTransaction([]string{key})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, failedTxn)
	ok, err = client.CompleteTransaction(txn, []storage.KeyValueData{{Key: key, Value: "unexpected"}})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, ok)
	ok, err = client.ExecuteTransaction(storage.CASRequest{
		ConditionKeys: []string{key}, RetryTillSuccessOrError: true,
		Update: func([]storage.KeyValueData) ([]storage.KeyValueData, bool, error) {
			t.Fatal("update must not run when reading the snapshot fails")
			return nil, false, nil
		},
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, ok)

	value, present, err = suite.client.Get(key)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "original", value)
}

func (suite *EtcdTestSuite) TestExecuteTransactionCommitErrorStopsRetries() {
	t := suite.T()
	key := t.Name()
	require.NoError(t, suite.client.Put(key, "original"))
	client := *suite.client
	calls := 0
	ok, err := client.ExecuteTransaction(storage.CASRequest{
		ConditionKeys: []string{key}, RetryTillSuccessOrError: true,
		Update: func([]storage.KeyValueData) ([]storage.KeyValueData, bool, error) {
			calls++
			require.Equal(t, 1, calls)
			client.timeout = -time.Second
			return []storage.KeyValueData{{Key: key, Value: "unexpected"}}, true, nil
		},
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, ok)
	require.Equal(t, 1, calls)
	value, present, err := suite.client.Get(key)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "original", value)
}

func (suite *EtcdTestSuite) TestClientInitializationErrors() {
	for _, tc := range []struct {
		name, settings, errorText string
	}{
		{"invalid timeout", `{"request_timeout":"invalid"}`, "request_timeout"},
		{"expired health probe", `{"request_timeout":"-1s"}`, "etcd not healthy"},
	} {
		suite.T().Run(tc.name, func(t *testing.T) {
			vip := readConfig(t, `{"payment_channel_storage_client":`+tc.settings+`}`)
			client, err := NewEtcdClientFromVip(vip, suite.metaData)
			require.ErrorContains(t, err, tc.errorText)
			require.Nil(t, client)
		})
	}
}

func (suite *EtcdTestSuite) TestClientHotReloadConfiguration() {
	for _, setting := range []string{"true", "false"} {
		suite.T().Run(setting, func(t *testing.T) {
			vip := readConfig(t, `{"payment_channel_storage_client":{"hot_reload":`+setting+`}}`)
			client, err := NewEtcdClientFromVip(vip, suite.metaData)
			require.NoError(t, err)
			t.Cleanup(client.Close)
			require.Equal(t, setting == "true", client.IsHotReloadEnabled())
		})
	}
}

func (suite *EtcdTestSuite) TestReconnect() {
	t := suite.T()
	client, err := Reconnect(suite.metaData)
	require.NoError(t, err)
	defer client.Close()
	require.NotSame(t, suite.client, client)
	key := t.Name()
	require.NoError(t, client.Put(key, "reconnected"))
	value, present, err := suite.client.Get(key)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "reconnected", value)
}

func TestInvalidEtcdServerConfiguration(t *testing.T) {
	vip := readConfig(t, `{"payment_channel_storage_server":{"startup_timeout":"invalid"}}`)
	_, err := GetEtcdServerConf(vip)
	require.ErrorContains(t, err, "startup_timeout")
	enabled, err := IsEtcdServerEnabledInVip(vip)
	require.ErrorContains(t, err, "startup_timeout")
	require.False(t, enabled)
	server, err := GetEtcdServerFromVip(vip)
	require.ErrorContains(t, err, "startup_timeout")
	require.Nil(t, server)
}

func TestEtcdServerStartRejectsInvalidConfiguration(t *testing.T) {
	conf, err := GetEtcdServerConf(config.Vip())
	require.NoError(t, err)
	conf.DataDir = t.TempDir()
	conf.Scheme = "invalid"
	conf.LogOutputs = []string{"stderr"}
	server := &EtcdServer{conf: conf}
	require.Error(t, server.Start())
	require.Nil(t, server.etcd)
}

func TestCloseUninitializedEtcdClient(t *testing.T) {
	require.NotPanics(t, func() { (&EtcdClient{}).Close() })
}

func Test_checkIfHttps(t *testing.T) {
	endpoint := []string{"https://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t, utils.CheckIfHttps(endpoint), true)
	endpoint = []string{"http://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t, utils.CheckIfHttps(endpoint), false)
}
