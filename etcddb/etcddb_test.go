package etcddb

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/escrow"
	"math/big"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

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
	var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	suite.metaData, _ = blockchain.InitOrganizationMetaDataFromJson(testJsonOrgGroupData)

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

	workDir := suite.server.conf.DataDir
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
			EtcdKeyValue{key: key1, value: expect1},
			EtcdKeyValue{key: key2, value: expect2},
		},
		[]EtcdKeyValue{
			EtcdKeyValue{key: key2, value: update2},
			EtcdKeyValue{key: key3, value: update3},
		},
	)
	assert.Nil(t, err)
	assert.True(t, ok)

	assertGet(suite, key1, expect1)
	assertGet(suite, key2, update2)
	assertGet(suite, key3, update3)

	ok, err = client.Transaction(
		[]EtcdKeyValue{
			EtcdKeyValue{key: key1, value: expect1},
			EtcdKeyValue{key: key2, value: expect2},
		},
		[]EtcdKeyValue{
			EtcdKeyValue{key: key2, value: update2},
			EtcdKeyValue{key: key3, value: update3},
		},
	)
	assert.Nil(t, err)
	assert.False(t, ok)

	assertGet(suite, key1, expect1)
	assertGet(suite, key2, update2)
	assertGet(suite, key3, update3)

	ok, err = client.Transaction(
		[]EtcdKeyValue{
			EtcdKeyValue{key: key1, value: expect1},
			EtcdKeyValue{key: key2, value: update2},
			EtcdKeyValue{key: key3, value: update3},
		},
		[]EtcdKeyValue{
			EtcdKeyValue{key: key2, value: expect2},
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

func removeWorkDir(t *testing.T, workDir string) {

	dir, err := os.Getwd()
	assert.Nil(t, err)

	err = os.RemoveAll(dir + "/" + workDir)
	assert.Nil(t, err)
}

func serialize(value interface{}) (slice string) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	err := e.Encode(value)
	if err != nil {
		return ""
	}

	slice = string(b.Bytes())
	return
}

func deserialize(slice string, value interface{}) (err error) {
	b := bytes.NewBuffer([]byte(slice))
	d := gob.NewDecoder(b)
	err = d.Decode(value)
	return
}
func (suite *EtcdTestSuite) TestCAS() {
	t := suite.T()
	client := suite.client

	usedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(1), Amount: big.NewInt(4), UsageType: escrow.USED_AMOUNT}
	plannedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(1), Amount: big.NewInt(10), UsageType: escrow.PLANNED_AMOUNT}
	failedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(1), Amount: big.NewInt(4), UsageType: escrow.REFUND_AMOUNT}
	err := client.Put("5", "val1")
	_ = client.Put(plannedData.Key(), serialize(plannedData))
	_ = client.Put(usedData.Key(), serialize(usedData))
	_ = client.Put(failedData.Key(), serialize(failedData))
	assert.Nil(t, err)
	request := &escrow.CASRequest{
		KeyPrefix:               "1",
		Condition:               escrow.IncrementUsedAmount,
		Action:                  escrow.BuildOldAndNewValuesForCAS,
		AdditionalParameters:    usedData,
		RetryTillSuccessOrError: true,
	}
	response, err := client.CAS(request)

	//assert.NotNil(t, err.Error(), "Usage Exceeded on channel 1")
	assert.Nil(t, err)
	assert.True(t, response.Succeeded)

}

func (suite *EtcdTestSuite) TestPlannedUsageCAS() {
	t := suite.T()
	client := suite.client

	plannedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(1), Amount: big.NewInt(10), UsageType: escrow.PLANNED_AMOUNT}
	assert.Equal(t, plannedData.ChannelID, big.NewInt(1))
	request := &escrow.CASRequest{
		KeyPrefix:               "3",
		Condition:               escrow.IncrementPlannedAmount,
		Action:                  escrow.BuildOldAndNewValuesForCAS,
		AdditionalParameters:    &escrow.PrePaidDataUnit{ChannelID: big.NewInt(3), Amount: big.NewInt(20)},
		RetryTillSuccessOrError: true,
	}
	response, err := client.CAS(request)

	//assert.NotNil(t, err.Error(), "Usage Exceeded on channel 1")
	assert.Nil(t, err)
	assert.True(t, response.Succeeded)
	value, ok, err := client.Get("3/P")
	data := &escrow.PrePaidDataUnit{}
	deserialize(value, data)
	assert.True(t, ok)
	assert.Equal(t, data.Amount, big.NewInt(20))

}

func (suite *EtcdTestSuite) TestEtcdCas() {
	t := suite.T()
	client := suite.client

	plannedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(1), Amount: big.NewInt(10), UsageType: escrow.PLANNED_AMOUNT}
	assert.Equal(t, plannedData.ChannelID, big.NewInt(1))
	request := &escrow.CASRequest{
		KeyPrefix:    "NoKey",
		NewKeyValues: []*escrow.KeyValueData{&escrow.KeyValueData{Key: "NewKey", Value: "NewValue"}},
	}
	response, err := client.etcdCas(request)

	//assert.NotNil(t, err.Error(), "Usage Exceeded on channel 1")
	assert.Nil(t, err)
	assert.True(t, response.Succeeded)
	value, ok, err := client.Get("NewKey")
	assert.True(t, ok)
	assert.Equal(t, value, "NewValue")

}

func (suite *EtcdTestSuite) TestCASConcurrentUpdate() {
	t := suite.T()
	client := suite.client

	usedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(6), Amount: big.NewInt(4), UsageType: escrow.USED_AMOUNT}
	plannedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(6), Amount: big.NewInt(10), UsageType: escrow.PLANNED_AMOUNT}
	failedData := &escrow.PrePaidDataUnit{ChannelID: big.NewInt(6), Amount: big.NewInt(4), UsageType: escrow.REFUND_AMOUNT}

	_ = client.Put(plannedData.Key(), serialize(plannedData))

	n := 7
	var start sync.WaitGroup

	start.Add(n)

	concurrentRequests := func(i int) {
		request := &escrow.CASRequest{
			KeyPrefix:               "6",
			Condition:               escrow.IncrementUsedAmount,
			Action:                  escrow.BuildOldAndNewValuesForCAS,
			AdditionalParameters:    usedData,
			RetryTillSuccessOrError: true,
		}
		client.CAS(request)
		defer start.Done()
	}

	for i := 0; i < n; i++ {
		go concurrentRequests(i)
	}
	start.Wait()

	//Now Add data for failed service calls
	_ = client.Put(failedData.Key(), serialize(failedData))
	value, ok, err := client.Get(usedData.Key())
	assert.True(t, ok)
	latestUsage := &escrow.PrePaidDataUnit{}
	err = deserialize(value, latestUsage)
	assert.Nil(t, err)
	assert.Equal(t, latestUsage.Amount, big.NewInt(8))
	_ = client.Put(failedData.Key(), serialize(failedData))

	assert.Nil(t, err)
	start.Add(n)
	for i := 0; i < n; i++ {
		go concurrentRequests(i)
	}
	start.Wait()

	value, ok, err = client.Get(usedData.Key())
	assert.True(t, ok)
	assert.Nil(t, err)
	latestUsage = &escrow.PrePaidDataUnit{}
	err = deserialize(value, latestUsage)
	assert.Nil(t, err)
	//Price is 4
	//Last Planned Amount was 10, Last Refund Amount was 4, last used amount was 8 , usage is 4
	// max usage that can be allowed is 12
	// 8 + x*4 <= 10 + 4 , so even though there were multiple threads only 1 could make it !!!!
	assert.Equal(t, latestUsage.Amount, big.NewInt(12))
}
func (suite *EtcdTestSuite) TestConcurrencyForDifferentCasUpdates() {
	t := suite.T()
	client := suite.client

	increaseUsedAmount := &escrow.CASRequest{
		KeyPrefix:               "8",
		Condition:               escrow.IncrementUsedAmount,
		Action:                  escrow.BuildOldAndNewValuesForCAS,
		AdditionalParameters:    &escrow.PrePaidDataUnit{ChannelID: big.NewInt(8), Amount: big.NewInt(1)},
		RetryTillSuccessOrError: true,
	}

	increasePlannedAmount := &escrow.CASRequest{
		KeyPrefix:               "8",
		Condition:               escrow.IncrementPlannedAmount,
		Action:                  escrow.BuildOldAndNewValuesForCAS,
		AdditionalParameters:    &escrow.PrePaidDataUnit{ChannelID: big.NewInt(8), Amount: big.NewInt(4)},
		RetryTillSuccessOrError: true,
	}

	increaseRefundAmount := &escrow.CASRequest{
		KeyPrefix:               "8",
		Condition:               escrow.IncrementRefundAmount,
		Action:                  escrow.BuildOldAndNewValuesForCAS,
		AdditionalParameters:    &escrow.PrePaidDataUnit{ChannelID: big.NewInt(8), Amount: big.NewInt(1)},
		RetryTillSuccessOrError: true,
	}
	n := 5
	var start sync.WaitGroup

	start.Add(n)
	client.CAS(increasePlannedAmount)
	concurrentRequests := func(i int) {

		if i%4 == 0 {
			client.CAS(increasePlannedAmount)
		}
		if i%3 == 0 {
			defer client.CAS(increaseRefundAmount)
		}

		defer func() {
			client.CAS(increaseUsedAmount)
			start.Done()

		}()

	}

	for i := 0; i < n; i++ {
		go concurrentRequests(i)
	}
	start.Wait()

	//Now Add data for failed service calls

	usedValue, ok, err := client.Get("8/U")
	assert.True(t, ok)
	usedAmt := &escrow.PrePaidDataUnit{}
	err = deserialize(usedValue, usedAmt)

	planedValue, ok, err := client.Get("8/P")
	assert.True(t, ok)
	plndAmt := &escrow.PrePaidDataUnit{}
	err = deserialize(planedValue, plndAmt)

	refundValue, ok, err := client.Get("8/R")
	assert.True(t, ok)
	refundAmt := &escrow.PrePaidDataUnit{}
	err = deserialize(refundValue, refundAmt)

	assert.Nil(t, err)
	assert.True(t, usedAmt.Amount.Cmp(plndAmt.Amount.Add(plndAmt.Amount, refundAmt.Amount)) <= 0)

}
