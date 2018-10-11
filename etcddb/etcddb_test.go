package etcddb

import (
	"fmt"
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type PaymentChannelStorageConf struct {
	ID         string
	ClientPort int `mapstructure:"CLIENT_PORT"`
	PeerPort   int `mapstructure:"PEER_PORT"`
	Token      string
}

func TestConfigParsing(t *testing.T) {

	const conf = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage_1=http://127.0.0.1:2480",

		"PAYMENT_CHANNEL_STORAGE": {
			"ID": "storage_1",
			"CLIENT_PORT": 2379,
			"PEER_PORT": 2380,
			"TOKEN": "payment_channel_storage_token"
		}
	}`

	vip := readConfig(t, conf)

	cluster := vip.GetString("PAYMENT_CHANNEL_STORAGE_CLUSTER")
	assert.Equal(t, "storage_1=http://127.0.0.1:2480", cluster)

	var paymentChannelStorageConf = PaymentChannelStorageConf{}
	err := vip.UnmarshalKey("PAYMENT_CHANNEL_STORAGE", &paymentChannelStorageConf)

	if err != nil {
		fmt.Println(err)
	}

	assert.Nil(t, err)
	assert.NotNil(t, paymentChannelStorageConf)

	assert.Equal(t, "storage_1", paymentChannelStorageConf.ID)
	assert.Equal(t, 2379, paymentChannelStorageConf.ClientPort)
	assert.Equal(t, 2380, paymentChannelStorageConf.PeerPort)
	assert.Equal(t, "payment_channel_storage_token", paymentChannelStorageConf.Token)
}

func readConfig(t *testing.T, configJSON string) (vip *viper.Viper) {
	vip = viper.New()
	err := config.ReadConfigFromJsonString(vip, configJSON)
	assert.Nil(t, err)
	return
}
