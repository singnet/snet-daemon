package etcddb

import (
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestEmptyPaymentChannelStorageConf(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2380"
	}`

	vip := readConfig(t, confJSON)

	cluster := GetPaymentChannelCluster(vip)
	assert.Equal(t, "storage_1=http://127.0.0.1:2380", cluster)

	conf, err := GetPaymentChannelStorageConf(vip)
	assert.Nil(t, err)
	assert.Nil(t, conf)

}

func TestConfigParsing(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2380",

		"PAYMENT_CHANNEL_STORAGE": {
			"ID": "storage_1",
			"CLIENT_PORT": 2379,
			"PEER_PORT": 2380,
			"TOKEN": "payment_channel_storage_token",
			"ENABLED": true
		}
	}`

	vip := readConfig(t, confJSON)

	cluster := GetPaymentChannelCluster(vip)
	assert.Equal(t, "storage_1=http://127.0.0.1:2380", cluster)

	conf, err := GetPaymentChannelStorageConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, "storage_1", conf.ID)
	assert.Equal(t, 2379, conf.ClientPort)
	assert.Equal(t, 2380, conf.PeerPort)
	assert.Equal(t, "payment_channel_storage_token", conf.Token)
	assert.Equal(t, true, conf.Enabled)
}

func TestPaymentChannelStorageConf(t *testing.T) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2380",

		"PAYMENT_CHANNEL_STORAGE": {
			"ID": "storage_1",
			"HOST" : "127.0.0.1",			
			"CLIENT_PORT": 2379,
			"PEER_PORT": 2380,
			"TOKEN": "payment_channel_storage_token",
			"ENABLED": true
		}
	}`

	vip := readConfig(t, confJSON)

	etcd, err := InitEtcdServer(vip)

	assert.Nil(t, err)
	assert.NotNil(t, etcd)

	defer etcd.Close()
}

func readConfig(t *testing.T, configJSON string) (vip *viper.Viper) {
	vip = viper.New()
	err := config.ReadConfigFromJsonString(vip, configJSON)
	assert.Nil(t, err)
	return
}
