package etcddb

import (
	"encoding/json"
	"strings"

	"github.com/spf13/viper"
)

const (
	// PaymentChannelStorageClientKey key for viper
	PaymentChannelStorageClientKey = "payment_channel_storage_client"

	// DefaultPaymentChannelStorageClientConf default client conf
	DefaultPaymentChannelStorageClientConf = `
	{
        "CONNECTION_TIMEOUT": 5000,
        "REQUEST_TIMEOUT": 3000
    }`

	// PaymentChannelStorageServerKey key for viper
	PaymentChannelStorageServerKey = "payment_channel_storage_server"

	// DefaultPaymentChannelStorageServerConf default server conf.
	// Note that running snet-daemon with several replicas require
	// that the default configuration file should be updated
	// according to the real information about etcd nodes in cluster.
	// Because DefaultPaymentChannelStorageServerConf is used when
	// the PAYMENT_CHANNEL_STORAGE_SERVER field is not set in the
	// property file it is treated as etcd server is not configured
	// and the ENABLED in this case is set to false by default.
	DefaultPaymentChannelStorageServerConf = `
	{
        "ID": "storage-1",
        "HOST" : "127.0.0.1",
        "CLIENT_PORT": 2379,
        "PEER_PORT": 2380,
        "TOKEN": "unique-token",
        "ENABLED": false
    }`
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
// The DefaultPaymentChannelStorageServerConf is used in case the PAYMENT_CHANNEL_STORAGE_SERVER field
// is not set in the configuration file
func GetPaymentChannelStorageServerConf(vip *viper.Viper) (conf *PaymentChannelStorageServerConf, err error) {

	conf = &PaymentChannelStorageServerConf{Scheme: "http", ClientPort: 2379, PeerPort: 2380, Enabled: true}

	if vip.Get(PaymentChannelStorageServerKey) == nil {
		defaultConf := normalizeDefaultConf(DefaultPaymentChannelStorageServerConf)
		err = json.Unmarshal([]byte(defaultConf), conf)
		return
	}

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
// The DefaultPaymentChannelStorageClientConf is used in case the PAYMENT_CHANNEL_STORAGE_CLIENT field
// is not set in the configuration file
func GetPaymentChannelStorageClientConf(vip *viper.Viper) (conf *PaymentChannelStorageClientConf, err error) {

	conf = &PaymentChannelStorageClientConf{ConnectionTimeout: 5, RequestTimeout: 3}

	if vip.Get(PaymentChannelStorageClientKey) == nil {
		defaultConf := normalizeDefaultConf(DefaultPaymentChannelStorageClientConf)
		err = json.Unmarshal([]byte(defaultConf), conf)
		return
	}

	err = vip.UnmarshalKey(PaymentChannelStorageClientKey, conf)

	return
}

func normalizeDefaultConf(conf string) string {
	return strings.Replace(conf, "_", "", -1)
}
