package etcddb

import (
	"github.com/spf13/viper"
)

const (
	// PaymentChannelStorageServerKey key for viper
	PaymentChannelStorageServerKey = "payment_channel_storage_server"

	// PaymentChannelStorageClientKey key for viper
	PaymentChannelStorageClientKey = "payment_channel_storage_client"
)

// PaymentChannelStorageServerConf contains embedded etcd server config
// ID - unique name of the etcd server node
// Scheme - URL schema used to create client and peer and urls
// Host - host where the etcd server is executed
// ClientPort - port to listen clients, used together with
//              Schema and host to compose listen-client-urls (see link below)
// PeerPort - port to listen etcd peers, used together with
//              Schema and host to compose listen-client-urls (see link below)
// Token - unique initial cluster token. Using unique token etcd can generate unique
//         cluster IDs and member IDs for the clusters even if they otherwise have
//         the exact same configuration. This can protect etcd from
//         cross-cluster-interaction, which might corrupt the clusters.
// Enabled - enable running embedded etcd server
// For more details see etcd Clustering Guide link:
// https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/clustering.md
type PaymentChannelStorageServerConf struct {
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

// GetPaymentChannelStorageServerConf gets PaymentChannelStorageServerConf from viper
func GetPaymentChannelStorageServerConf(vip *viper.Viper) (conf *PaymentChannelStorageServerConf, err error) {

	if vip.Get(PaymentChannelStorageServerKey) == nil {
		return
	}

	conf = &PaymentChannelStorageServerConf{Scheme: "http", ClientPort: 2379, PeerPort: 2380, Enabled: true}
	err = vip.UnmarshalKey(PaymentChannelStorageServerKey, conf)
	return
}

// PaymentChannelStorageClientConf config
// ConnectionTimeout - timeout for failing to establish a connection
// RequestTimeout    - per request timeout
type PaymentChannelStorageClientConf struct {
	ConnectionTimeout int `mapstructure:"CONNECTION_TIMEOUT"`
	RequestTimeout    int `mapstructure:"REQUEST_TIMEOUT"`
}

// GetPaymentChannelStorageClientConf gets PaymentChannelStorageServerConf from viper
func GetPaymentChannelStorageClientConf(vip *viper.Viper) (conf *PaymentChannelStorageClientConf, err error) {

	if vip.Get(PaymentChannelStorageClientKey) == nil {
		return
	}

	conf = &PaymentChannelStorageClientConf{}
	err = vip.UnmarshalKey(PaymentChannelStorageClientKey, conf)

	return
}
