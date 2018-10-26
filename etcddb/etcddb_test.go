package etcddb

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/smartystreets/gunit"
	"github.com/stretchr/testify/assert"
)

// TODO: initialize client and server only once to make test faster

var testingEtcdDB *testing.T

func TestEtcdWithGUnit(t *testing.T) {

	testingEtcdDB = t
	gunit.RunSequential(new(EtcdTestFixture), t)
}

type EtcdTestFixture struct {
	*gunit.Fixture
	client *EtcdClient
	server *EtcdServer
}

func (fixture *EtcdTestFixture) SetupStuff() {
	fmt.Println("SetupStuff")

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
			"enabled": true
		}
	}`

	t := testingEtcdDB
	vip := readConfig(t, confJSON)
	server, err := GetEtcdServerFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)
	fixture.server = server
	err = server.Start()
	assert.Nil(t, err)

	client, err := NewEtcdClientFromVip(vip)

	assert.Nil(t, err)
	assert.NotNil(t, client)
	fixture.client = client
}

func (fixture *EtcdTestFixture) TeardownStuff() {

	defer removeWorkDirs()

	if fixture.client != nil {
		fixture.client.Close()
	}

	if fixture.server != nil {
		fixture.server.Close()
	}

}

func (fixture *EtcdTestFixture) TestEtcdPutGet() {

	t := testingEtcdDB

	client := fixture.client
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

func (fixture *EtcdTestFixture) TestEtcdCAS() {

	t := testingEtcdDB
	client := fixture.client

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

func (fixture *EtcdTestFixture) TestEtcdNilValue() {

	t := testingEtcdDB
	client := fixture.client

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

func removeWorkDirs() {

	t := testingEtcdDB

	dir, err := os.Getwd()
	assert.Nil(t, err)

	files, err := ioutil.ReadDir(dir)
	assert.Nil(t, err)

	for _, f := range files {
		name := f.Name()
		if f.IsDir() && strings.HasPrefix(name, "storage-") {
			fmt.Println(name)
			err = os.RemoveAll(name)
			assert.Nil(t, err)
		}
	}
}
