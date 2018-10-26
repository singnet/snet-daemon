package etcddb

import (
	"testing"
	"time"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEtcdClientConf(t *testing.T) {

	conf, err := GetEtcdClientConf(config.Vip())

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, 5*time.Second, conf.ConnectionTimeout)
	assert.Equal(t, 3*time.Second, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2379"}, conf.Endpoints)
}

func TestCustomEtcdClientConf(t *testing.T) {

	const confJSON = `
	{
		"payment_channel_storage_client": {
			"connection_timeout": "15s",
			"request_timeout": "5s",
			"endpoints": ["http://127.0.0.1:2479"]
		}
	}`

	vip := readConfig(t, confJSON)

	conf, err := GetEtcdClientConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 15*time.Second, conf.ConnectionTimeout)
	assert.Equal(t, 5*time.Second, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2479"}, conf.Endpoints)
}
func TestCustomEtcdClientConfWithDefault(t *testing.T) {

	const confJSON = `
	{
		"payment_channel_storage_client": {
			"connection_timeout": "15s"
		}
	}`

	vip := readConfig(t, confJSON)

	conf, err := GetEtcdClientConf(vip)

	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 15*time.Second, conf.ConnectionTimeout)
	assert.Equal(t, 3*time.Second, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2379"}, conf.Endpoints)
}

func TestDefaultEtcdServerConf(t *testing.T) {

	enabled, err := IsEtcdServerEnabled()
	assert.Nil(t, err)
	assert.True(t, enabled)

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
	assert.Equal(t, time.Minute, conf.StartupTimeout)
	assert.Equal(t, true, conf.Enabled)

	server, err := GetEtcdServer()

	assert.Nil(t, err)
	assert.NotNil(t, server)
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
			"startup_timeout": "15s",
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
	assert.Equal(t, "http", conf.Scheme)
	assert.Equal(t, "127.0.0.1", conf.Host)
	assert.Equal(t, 2379, conf.ClientPort)
	assert.Equal(t, 2380, conf.PeerPort)
	assert.Equal(t, "unique-token", conf.Token)
	assert.Equal(t, 15*time.Second, conf.StartupTimeout)
	assert.Equal(t, true, conf.Enabled)

	server, err := GetEtcdServerFromVip(vip)
	assert.Nil(t, err)
	assert.NotNil(t, server)

	err = server.Start()
	assert.Nil(t, err)
	defer server.Close()
}

func readConfig(t *testing.T, configJSON string) (vip *viper.Viper) {
	vip = viper.New()
	config.SetDefaultFromConfig(vip, config.Vip())

	err := config.ReadConfigFromJsonString(vip, configJSON)
	assert.Nil(t, err)
	return
}
