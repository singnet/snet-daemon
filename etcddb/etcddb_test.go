package etcddb

import (
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetPaymentChannelCluster(t *testing.T) {
	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380"
	}`

	vip := readConfig(t, confJSON)

	cluster := GetPaymentChannelCluster(vip)
	assert.Equal(t, "storage-1=http://127.0.0.1:2380", cluster)
}

func TestDefaultEtcdServerConf(t *testing.T) {

	const confJSON = `{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380"
	}`

	vip := readConfig(t, confJSON)
	conf, err := GetPaymentChannelStorageServerConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, "storage-1", conf.ID)
	assert.Equal(t, "127.0.0.1", conf.Host)
	assert.Equal(t, 2379, conf.ClientPort)
	assert.Equal(t, 2380, conf.PeerPort)
	assert.Equal(t, "unique-token", conf.Token)
	assert.Equal(t, false, conf.Enabled)

	server, err := InitEtcdServer(vip)

	assert.Nil(t, err)
	assert.Nil(t, server)
}

func TestDisabledEtcdServerConf(t *testing.T) {

	const confJSON = `
		{
			"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380",

			"PAYMENT_CHANNEL_STORAGE_SERVER": {
				"ENABLED": false
			}
		}`

	vip := readConfig(t, confJSON)
	server, err := InitEtcdServer(vip)

	assert.Nil(t, err)
	assert.Nil(t, server)
}

func TestEnabledEtcdServerConf(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380",

		"PAYMENT_CHANNEL_STORAGE_SERVER": {
			"ID": "storage-1",
			"HOST" : "127.0.0.1",
			"CLIENT_PORT": 2379,
			"PEER_PORT": 2380,
			"TOKEN": "unique-token",
			"ENABLED": true
		}
	}`

	vip := readConfig(t, confJSON)
	cluster := GetPaymentChannelCluster(vip)
	assert.Equal(t, "storage-1=http://127.0.0.1:2380", cluster)

	conf, err := GetPaymentChannelStorageServerConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, "storage-1", conf.ID)
	assert.Equal(t, "127.0.0.1", conf.Host)
	assert.Equal(t, 2379, conf.ClientPort)
	assert.Equal(t, 2380, conf.PeerPort)
	assert.Equal(t, "unique-token", conf.Token)
	assert.Equal(t, true, conf.Enabled)

	server, err := InitEtcdServer(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)
	defer server.Close()
}

func TestDefaultEtcdClientConf(t *testing.T) {

	const confJSON = `{}`
	vip := readConfig(t, confJSON)
	conf, err := GetPaymentChannelStorageClientConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, 5000, conf.ConnectionTimeout)
	assert.Equal(t, 3000, conf.RequestTimeout)
}

func TestEtcdClientConf(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380",

		"PAYMENT_CHANNEL_STORAGE_CLIENT": {
			"CONNECTION_TIMEOUT": 15000,
			"REQUEST_TIMEOUT": 5000
		}
	}`

	vip := readConfig(t, confJSON)
	cluster := GetPaymentChannelCluster(vip)
	assert.Equal(t, "storage-1=http://127.0.0.1:2380", cluster)

	conf, err := GetPaymentChannelStorageClientConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 15000, conf.ConnectionTimeout)
	assert.Equal(t, 5000, conf.RequestTimeout)
}

func TestPaymentChannelStorageReadWrite(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380",

		"PAYMENT_CHANNEL_STORAGE_CLIENT": {
			"CONNECTION_TIMEOUT": 5000,
			"REQUEST_TIMEOUT": 3000
		},

		"PAYMENT_CHANNEL_STORAGE_SERVER": {
			"ID": "storage-1",
			"HOST" : "127.0.0.1",
			"CLIENT_PORT": 2379,
			"PEER_PORT": 2380,
			"TOKEN": "unique-token",
			"ENABLED": true
		}
	}`

	vip := readConfig(t, confJSON)

	server, err := InitEtcdServer(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)

	defer server.Close()

	client, err := NewEtcdClient(vip)

	assert.Nil(t, err)
	assert.NotNil(t, client)
	defer client.Close()

	key := "key"
	value := "value"
	keyBytes := stringToByteArray(key)
	valueBytes := stringToByteArray(value)

	err = client.Put(keyBytes, valueBytes)
	assert.Nil(t, err)

	getResult, err := client.Get(keyBytes)
	assert.Nil(t, err)
	assert.True(t, len(getResult) > 0)
	assert.Equal(t, value, byteArraytoString(getResult))

	err = client.Delete(keyBytes)
	assert.Nil(t, err)

	getResult, err = client.Get(keyBytes)
	assert.Nil(t, err)
	assert.True(t, len(getResult) == 0)

}

func TestPaymentChannelStorageCAS(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380",

		"PAYMENT_CHANNEL_STORAGE_SERVER": {
			"ID": "storage-1",
			"HOST" : "127.0.0.1",
			"TOKEN": "unique-token"
		}
	}`

	vip := readConfig(t, confJSON)

	server, err := InitEtcdServer(vip)

	assert.Nil(t, err)
	assert.NotNil(t, server)

	defer server.Close()

	client, err := NewEtcdClient(vip)

	assert.Nil(t, err)
	assert.NotNil(t, client)

	defer client.Close()

	key := "key"
	expect := "expect"
	update := "update"
	keyBytes := stringToByteArray(key)
	expectBytes := stringToByteArray(expect)
	updateBytes := stringToByteArray(update)

	err = client.Put(keyBytes, expectBytes)
	assert.Nil(t, err)

	ok, err := client.CompareAndSet(
		keyBytes,
		expectBytes,
		updateBytes,
	)
	assert.Nil(t, err)
	assert.True(t, ok)

	updateResult, err := client.Get(keyBytes)
	assert.Nil(t, err)
	assert.Equal(t, update, byteArraytoString(updateResult))

	ok, err = client.CompareAndSet(
		keyBytes,
		expectBytes,
		updateBytes,
	)
	assert.Nil(t, err)
	assert.False(t, ok)
}
func TestMissedEndpoints(t *testing.T) {

	endpoints, err := getPaymentChannelEndpoints("")
	assert.NotNil(t, err)
	assert.Nil(t, endpoints)
}

func TestGetClusterEndpoints(t *testing.T) {

	checkGetClusterEndpoints(t, "storage-1=http://127.0.0.1:2380", []string{"http://127.0.0.1:2380"})
	checkGetClusterEndpoints(
		t,
		"storage-1=http://127.0.0.1:2380,storage-2=http://127.0.0.1:2480",
		[]string{"http://127.0.0.1:2380", "http://127.0.0.1:2480"},
	)
}

func checkGetClusterEndpoints(t *testing.T, cluster string, expects []string) {

	endpoints, err := getPaymentChannelEndpoints(cluster)

	assert.Nil(t, err)
	assert.Equal(t, expects, endpoints)
}

func readConfig(t *testing.T, configJSON string) (vip *viper.Viper) {
	vip = viper.New()
	err := config.ReadConfigFromJsonString(vip, configJSON)
	assert.Nil(t, err)
	return
}
