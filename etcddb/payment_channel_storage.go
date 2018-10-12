package etcddb

import (
	"github.com/spf13/viper"
)

const (
	// PaymentChannelStorageKey key for viper
	PaymentChannelStorageKey = "payment_channel_storage"

	// PaymentChannelStorageClusterKey key for viper
	PaymentChannelStorageClusterKey = "payment_channel_storage_cluster"
)

// PaymentChannelStorageConf contains embedded etcd server conf
type PaymentChannelStorageConf struct {
	ID         string
	Scheme     string
	Host       string
	ClientPort int `mapstructure:"CLIENT_PORT"`
	PeerPort   int `mapstructure:"PEER_PORT"`
	Token      string
	Enabled    bool
}

// GetPaymentChannelCluster gets payment channel cluster from viper
func GetPaymentChannelCluster(vip *viper.Viper) string {
	return vip.GetString("PAYMENT_CHANNEL_STORAGE_CLUSTER")
}

// GetPaymentChannelStorageConf gets PaymentChannelStorageConf from viper
func GetPaymentChannelStorageConf(vip *viper.Viper) (conf *PaymentChannelStorageConf, err error) {

	if vip.Get(PaymentChannelStorageKey) == nil {
		return
	}

	conf = &PaymentChannelStorageConf{Scheme: "http", Enabled: true}
	err = vip.UnmarshalKey(PaymentChannelStorageKey, conf)
	return
}
