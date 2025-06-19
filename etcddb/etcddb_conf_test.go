package etcddb

import (
	"os"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEtcdClientConf(t *testing.T) {

	var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	metadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	conf, err := GetEtcdClientConf(config.Vip(), metadata)

	assert.Nil(t, err)
	assert.NotNil(t, conf)

	assert.Equal(t, 5*time.Second, conf.ConnectionTimeout)
	assert.Equal(t, 3*time.Second, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2379"}, conf.Endpoints)
}

func TestCustomEtcdClientConf(t *testing.T) {
	var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"5s\",           \"endpoints\": [             \"http://127.0.0.1:2479\"           ]         }       }     },      {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	metadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))

	conf, err := GetEtcdClientConf(nil, metadata)

	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 15*time.Second, conf.ConnectionTimeout)
	assert.Equal(t, 5*time.Second, conf.RequestTimeout)
	assert.Equal(t, []string{"http://127.0.0.1:2479"}, conf.Endpoints)
}

func TestCustomEtcdClientConfWithDefault(t *testing.T) {
	var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",                    \"endpoints\": [             \"http://127.0.0.1:2479\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"5s\",           \"request_timeout\": \"3s\"                 }       }     }   ] }"
	metadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	assert.Nil(t, metadata)
	assert.NotNil(t, err)

	if metadata != nil {
		conf, err := GetEtcdClientConf(nil, metadata)
		assert.NotNil(t, err)
		assert.Nil(t, conf)
	}

}

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
	assert.Equal(t, time.Minute, conf.StartupTimeout)
	assert.Equal(t, "storage-data-dir-1.etcd", conf.DataDir)
	assert.Equal(t, "info", conf.LogLevel)
	assert.Equal(t, []string{"./etcd-server.log"}, conf.LogOutputs)
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
			"startup_timeout": "15s",
			"data_dir": "custom-storage-data-dir-1.etcd",
			"log_level": "warn",
			"log_outputs": ["stderr"],
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
	assert.Equal(t, "custom-storage-data-dir-1.etcd", conf.DataDir)
	assert.Equal(t, "warn", conf.LogLevel)
	assert.Equal(t, []string{"stderr"}, conf.LogOutputs)
	assert.Equal(t, true, conf.Enabled)

	server, err := GetEtcdServerFromVip(vip)
	assert.Nil(t, err)
	assert.NotNil(t, server)

	err = server.Start()
	assert.Nil(t, err)
	defer removeWorkDir(t, conf.DataDir)
	defer server.Close()
}

func readConfig(t *testing.T, configJSON string) (vip *viper.Viper) {
	vip = viper.New()
	config.SetDefaultFromConfig(vip, config.Vip())
	err := config.ReadConfigFromJsonString(vip, configJSON)
	assert.Nil(t, err)
	return
}

func removeWorkDir(t *testing.T, workDir string) {
	dir, err := os.Getwd()
	assert.Nil(t, err)
	err = os.RemoveAll(dir + "/" + workDir)
	assert.Nil(t, err)
}
