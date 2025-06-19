package etcddb

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TODO: initialize client and server only once to make test faster

type EtcdTestSuite struct {
	suite.Suite
	client   *EtcdClient
	server   *EtcdServer
	metaData *blockchain.OrganizationMetaData
}

func TestEtcdTestSuite(t *testing.T) {
	suite.Run(t, new(EtcdTestSuite))
}

func (suite *EtcdTestSuite) BeforeTest(suiteName string, testName string) {
	var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"500s\",           \"request_timeout\": \"300s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	suite.metaData, _ = blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))

	const confJSON = `
	{
		"payment_channel_storage_client": {
			"connection_timeout": "5s",
			"request_timeout": "3s",
			"endpoints": ["http://127.0.0.1:2379"]
		},

		"payment_channel_storage_server": {
			"id": "storage-1",
			"host" : "127.0.0.1",
			"client_port": 2379,
			"peer_port": 2380,
			"token": "unique-token",
			"cluster": "storage-1=http://127.0.0.1:2380",
			"data_dir": "storage-data-dir-1.etcd",
			"enabled": true
		}
	}`

	t := suite.T()
	vip := readConfig(t, confJSON)
	server, err := GetEtcdServerFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)
	suite.server = server

	err = server.Start()
	assert.Nil(t, err)

	client, err := NewEtcdClientFromVip(vip, suite.metaData)

	assert.Nil(t, err)
	assert.NotNil(t, client)
	suite.client = client

}

func (suite *EtcdTestSuite) AfterTest(suiteName string, testName string) {

	workDir := suite.server.GetConf().DataDir
	defer removeWorkDir(suite.T(), workDir)

	if suite.client != nil {
		suite.client.Close()
	}

	if suite.server != nil {
		suite.server.Close()
	}

}

func (suite *EtcdTestSuite) TestEtcdPutGet() {

	t := suite.T()

	client := suite.client
	missedValue, ok, err := client.Get("missed_key")
	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", missedValue)

	key := "key"
	value := "value"

	err = client.Put(key, value)
	assert.Nil(t, err)

	getResult, ok, err := client.Get(key)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.True(t, len(getResult) > 0)
	assert.Equal(t, value, getResult)

	err = client.Delete(key)
	assert.Nil(t, err)

	getResult, ok, err = client.Get(key)
	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", getResult)

	// GetWithRange
	count := 3
	keyValues := getKeyValuesWithPrefix("key-range-bbb-", "value-range", count)

	for _, keyValue := range keyValues {
		err = client.Put(keyValue.key, keyValue.value)
		assert.Nil(t, err)
	}

	err = client.Put("key-range-bba", "value-range-before")
	assert.Nil(t, err)
	err = client.Put("key-range-bbc", "value-range-after")
	assert.Nil(t, err)

	values, err := client.GetByKeyPrefix("key-range-bbb-")
	assert.Nil(t, err)
	assert.Equal(t, count, len(values))

	for index, value := range values {
		assert.Equal(t, keyValues[index].value, value)
	}
}

func (suite *EtcdTestSuite) TestEtcdCAS() {

	t := suite.T()
	client := suite.client

	key := "key"
	expect := "expect"
	update := "update"

	err := client.Put(key, expect)
	assert.Nil(t, err)

	ok, err := client.CompareAndSwap(
		key,
		expect,
		update,
	)
	assert.Nil(t, err)
	assert.True(t, ok)

	updateResult, ok, err := client.Get(key)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, update, updateResult)

	ok, err = client.CompareAndSwap(
		key,
		expect,
		update,
	)
	assert.Nil(t, err)
	assert.False(t, ok)
}
func (suite *EtcdTestSuite) TestEtcdTransaction() {

	t := suite.T()
	client := suite.client

	key1 := "key1"
	expect1 := "expect1"

	key2 := "key2"
	expect2 := "expect2"
	update2 := "update2"

	key3 := "key3"
	update3 := "update3"

	err := client.Put(key1, expect1)
	assert.Nil(t, err)

	err = client.Put(key2, expect2)
	assert.Nil(t, err)

	assertGet(suite, key1, expect1)
	assertGet(suite, key2, expect2)

	ok, err := client.Transaction(
		[]EtcdKeyValue{
			NewEtcdKeyValue(key1, expect1),
			NewEtcdKeyValue(key2, expect2),
		},
		[]EtcdKeyValue{
			NewEtcdKeyValue(key2, update2),
			NewEtcdKeyValue(key3, update3),
		},
	)
	assert.Nil(t, err)
	assert.True(t, ok)

	assertGet(suite, key1, expect1)
	assertGet(suite, key2, update2)
	assertGet(suite, key3, update3)

	ok, err = client.Transaction(
		[]EtcdKeyValue{
			NewEtcdKeyValue(key1, expect1),
			NewEtcdKeyValue(key2, expect2),
		},
		[]EtcdKeyValue{
			NewEtcdKeyValue(key2, update2),
			NewEtcdKeyValue(key3, update3),
		},
	)
	assert.Nil(t, err)
	assert.False(t, ok)

	assertGet(suite, key1, expect1)
	assertGet(suite, key2, update2)
	assertGet(suite, key3, update3)

	ok, err = client.Transaction(
		[]EtcdKeyValue{
			NewEtcdKeyValue(key1, expect1),
			NewEtcdKeyValue(key2, update2),
			NewEtcdKeyValue(key3, update3),
		},
		[]EtcdKeyValue{
			NewEtcdKeyValue(key2, expect2),
		},
	)
	assert.Nil(t, err)
	assert.True(t, ok)

	assertGet(suite, key1, expect1)
	assertGet(suite, key2, expect2)
	assertGet(suite, key3, update3)
}

func (suite *EtcdTestSuite) TestEtcdNilValue() {

	t := suite.T()
	client := suite.client

	key := "key-for-nil-value"

	err := client.Delete(key)
	assert.Nil(t, err)

	missedValue, ok, err := client.Get(key)

	assert.Nil(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", missedValue)

	err = client.Put(key, "")
	assert.Nil(t, err)

	nillValue, ok, err := client.Get(key)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, "", nillValue)

	err = client.Delete(key)
	assert.Nil(t, err)

	firstValue := "first-value"
	ok, err = client.PutIfAbsent(key, firstValue)
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = client.PutIfAbsent(key, firstValue)
	assert.Nil(t, err)
	assert.False(t, ok)

}

func (suite *EtcdTestSuite) TestPutIfAbsentTransactionFailsAfterConcurrentPut() {

	t := suite.T()
	client := suite.client

	// before
	key := "key-put-if-absent-transaction-fails-after-concurrent-put"
	_, ok, err := client.Get(key)
	assert.Nil(t, err)
	assert.False(t, ok)

	// when
	transaction, err := client.StartTransaction([]string{key})
	assert.Nil(t, err)

	err = client.Put(key, "concurrent-value")
	assert.Nil(t, err)

	// then
	ok, err = client.CompleteTransaction(transaction, []storage.KeyValueData{
		{
			Key:     key,
			Value:   "transaction-value",
			Present: true,
		}})
	assert.Nil(t, err)
	assert.False(t, ok)
}

func (suite *EtcdTestSuite) TestEtcdMutex() {

	t := suite.T()

	keyA := "key-a"
	keyB := "key-b"
	lockKey := "key-mutex"

	n := 7
	var start sync.WaitGroup
	var end sync.WaitGroup
	start.Add(n)
	end.Add(n)

	runWithLock := func(i int) {

		client, err := NewEtcdClient(suite.metaData)
		assert.Nil(t, err)
		defer client.Close()

		value := strconv.Itoa(i)

		mutex, err := client.NewMutex(lockKey)
		assert.Nil(t, err)
		defer mutex.Unlock(context.Background())
		defer end.Done()
		start.Done()
		start.Wait()

		err = mutex.Lock(context.Background())
		assert.Nil(t, err)

		err = client.Put(keyA, value)
		assert.Nil(t, err)

		time.Sleep(200 * time.Millisecond)

		err = client.Put(keyB, value)
		assert.Nil(t, err)
	}

	for i := 0; i < n; i++ {
		go runWithLock(i)
	}

	client := suite.client

	end.Wait()
	res1, ok, err := client.Get(keyA)
	assert.True(t, ok)
	assert.Nil(t, err)
	res2, ok, err := client.Get(keyB)
	assert.True(t, ok)
	assert.Nil(t, err)
	assert.Equal(t, res1, res2)
}

func assertGet(suite *EtcdTestSuite, key string, value string) {
	t := suite.T()
	updateResult, ok, err := suite.client.Get(key)
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, value, updateResult)
}

type keyValue struct {
	key   string
	value string
}

func getKeyValuesWithPrefix(keyPrefix string, valuePrefix string, count int) (keyValues []keyValue) {
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%s-%d", keyPrefix, i)
		value := fmt.Sprintf("%s-%d", valuePrefix, i)
		keyValue := keyValue{key, value}
		keyValues = append(keyValues, keyValue)
	}
	return
}
