package etcddb

import (
	"fmt"
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEtcdServerConf(t *testing.T) {

	enabled, err := IsEtcdServerEnabled()
	assert.Nil(t, err)
	assert.False(t, enabled)

	conf, err := GetEtcdServerConf(config.Vip())

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, "storage-1", conf.ID)
	assert.Equal(t, "http", conf.Scheme)
	assert.Equal(t, "127.0.0.1", conf.Host)
	assert.Equal(t, 2379, conf.ClientPort)
	assert.Equal(t, 2380, conf.PeerPort)
	assert.Equal(t, "unique-token", conf.Token)
	assert.Equal(t, "storage-1=http://127.0.0.1:2380", conf.Cluster)
	assert.Equal(t, false, conf.Enabled)

	server, err := GetEtcdServer()

	assert.Nil(t, err)
	assert.Nil(t, server)
}

func TestDisabledEtcdServerConf(t *testing.T) {

	const confJSON = `
		{
			"payment_channel_storage_server": {
				"enabled": false
			}
		}`

	vip := readConfig(t, confJSON)
	enabled, err := IsEtcdServerEnabledInVip(vip)
	assert.Nil(t, err)
	assert.False(t, enabled)

	server, err := GetEtcdServerFromVip(vip)

	assert.Nil(t, err)
	assert.Nil(t, server)
}

func TestEnabledEtcdServerConf(t *testing.T) {

	const confJSON = `
	{
		"payment_channel_storage_server": {
			"id": "storage-1",
			"host" : "127.0.0.1",
			"client_port": 2379,
			"peer_port": 2380,
			"token": "unique-token",
			"cluster": "storage-1=http://127.0.0.1:2380",
			"enabled": true
		}
	}`

	vip := readConfig(t, confJSON)

	enabled, err := IsEtcdServerEnabledInVip(vip)
	assert.Nil(t, err)
	assert.True(t, enabled)

	conf, err := GetEtcdServerConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, "storage-1", conf.ID)
	assert.Equal(t, "127.0.0.1", conf.Host)
	assert.Equal(t, 2379, conf.ClientPort)
	assert.Equal(t, 2380, conf.PeerPort)
	assert.Equal(t, "unique-token", conf.Token)
	assert.Equal(t, true, conf.Enabled)

	server, err := GetEtcdServerFromVip(vip)
	assert.Nil(t, err)
	assert.NotNil(t, server)

	err = server.Start()
	assert.Nil(t, err)
	defer server.Close()
}

func TestDefaultEtcdClientConf(t *testing.T) {

	conf, err := GetEtcdClientConf(config.Vip())

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, 5000, conf.ConnectionTimeout)
	assert.Equal(t, 3000, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2379"}, conf.Endpoints)
}

func TestEtcdClientConf(t *testing.T) {

	const confJSON = `
	{
		"payment_channel_storage_client": {
			"connection_timeout": 15000,
			"request_timeout": 5000,
			"endpoints": ["http://127.0.0.1:2479"]
		}
	}`

	vip := readConfig(t, confJSON)

	conf, err := GetEtcdClientConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 15000, conf.ConnectionTimeout)
	assert.Equal(t, 5000, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2479"}, conf.Endpoints)
}

func TestEtcdPutGet(t *testing.T) {

	const confJSON = `
	{
		"payment_channel_storage_client": {
			"connection_timeout": 5000,
			"request_timeout": 3000,
			"endpoints": ["http://127.0.0.1:2379"]
		},

		"payment_channel_storage_server": {
			"id": "storage-1",
			"host" : "127.0.0.1",
			"client_port": 2379,
			"peer_port": 2380,
			"token": "unique-token",
			"cluster": "storage-1=http://127.0.0.1:2380",
			"enabled": true
		}
	}`

	vip := readConfig(t, confJSON)

	server, err := GetEtcdServerFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)
	err = server.Start()
	assert.Nil(t, err)

	defer server.Close()

	client, err := NewEtcdClientFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, client)
	defer client.Close()

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

	values, ok, err := client.GetByKeyPrefix("key-range-bbb-")
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, count, len(values))

	for index, value := range values {
		assert.Equal(t, keyValues[index].value, value)
	}
}

func TestEtcdCAS(t *testing.T) {

	const confJSON = `
	{
		"payment_channel_storage_server": {
			"id": "storage-1",
			"host" : "127.0.0.1",
			"cluster": "storage-1=http://127.0.0.1:2380",
			"token": "unique-token"
		}
	}`

	vip := readConfig(t, confJSON)

	server, err := GetEtcdServerFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)

	err = server.Start()
	assert.Nil(t, err)
	defer server.Close()

	client, err := NewEtcdClient()

	assert.Nil(t, err)
	assert.NotNil(t, client)

	defer client.Close()

	key := "key"
	expect := "expect"
	update := "update"

	err = client.Put(key, expect)
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

func TestEtcdNilValue(t *testing.T) {

	const confJSON = `
	{ "payment_channel_storage_server": {} }`

	vip := readConfig(t, confJSON)

	server, err := GetEtcdServerFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)

	err = server.Start()
	assert.Nil(t, err)
	defer server.Close()

	client, err := NewEtcdClient()

	assert.Nil(t, err)
	assert.NotNil(t, client)
	defer client.Close()

	key := "key-for-nil-value"

	err = client.Delete(key)
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

func readConfig(t *testing.T, configJSON string) (vip *viper.Viper) {
	vip = viper.New()
	err := config.ReadConfigFromJsonString(vip, configJSON)
	assert.Nil(t, err)
	return
}
