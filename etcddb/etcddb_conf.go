package etcddb

import (
	"strings"
	"time"

	"github.com/coreos/pkg/capnslog"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// EtcdClientConf config
// ConnectionTimeout - timeout for failing to establish a connection
// RequestTimeout    - per request timeout
// Endpoints         - cluster endpoints
type EtcdClientConf struct {
	ConnectionTimeout time.Duration `json:"connection_timeout" mapstructure:"connection_timeout"`
	RequestTimeout    time.Duration `json:"request_timeout" mapstructure:"request_timeout"`
	Endpoints         []string
}

// GetEtcdClientConf gets EtcdServerConf from viper
// The DefaultEtcdClientConf is used in case the PAYMENT_CHANNEL_STORAGE_CLIENT field
// is not set in the configuration file
func GetEtcdClientConf(vip *viper.Viper) (conf *EtcdClientConf, err error) {

	key := config.PaymentChannelStorageClientKey
	conf = &EtcdClientConf{}
	subVip := config.SubWithDefault(vip, key)
	err = subVip.Unmarshal(conf)
	return
}

func normalizeDefaultConf(conf string) string {
	return strings.Replace(conf, "_", "", -1)
}

// EtcdServerConf contains embedded etcd server config
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
// StartupTimeout - time to wait the etcd server successfully started
// Enabled - enable running embedded etcd server
// For more details see etcd Clustering Guide link:
// https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/clustering.md
type EtcdServerConf struct {
	ID             string
	Scheme         string
	Host           string
	ClientPort     int `json:"client_port" mapstructure:"CLIENT_PORT"`
	PeerPort       int `json:"peer_port" mapstructure:"PEER_PORT"`
	Token          string
	Cluster        string
	StartupTimeout time.Duration `json:"startup_timeout" mapstructure:"startup_timeout"`
	Enabled        bool
	DataDir        string `json:"data_dir" mapstructure:"DATA_DIR"`
	LogLevel       string `json:"log_level" mapstructure:"LOG_LEVEL"`
}

// GetEtcdServerConf gets EtcdServerConf from viper
// The DefaultEtcdServerConf is used in case the PAYMENT_CHANNEL_STORAGE_SERVER field
// is not set in the configuration file
func GetEtcdServerConf(vip *viper.Viper) (conf *EtcdServerConf, err error) {

	key := config.PaymentChannelStorageServerKey
	conf = &EtcdServerConf{}

	subVip := config.SubWithDefault(vip, key)
	err = subVip.Unmarshal(conf)
	if err != nil {
		return
	}

	if !vip.InConfig(strings.ToLower(key)) {
		return
	}

	conf.Enabled = true

	err = vip.UnmarshalKey(key, conf)

	if err != nil {
		return
	}

	err = initEtcdLogger(conf)

	return
}

// capnslog to logrus formatter implementation
// with methods Format and Flush
type capnslogToLogrusLogFormatter struct {
}

func (formatter *capnslogToLogrusLogFormatter) Format(pkg string, level capnslog.LogLevel,
	depth int, entries ...interface{}) {

	l := log.WithField("pkg", pkg)

	switch level {
	case capnslog.CRITICAL, capnslog.ERROR:
		l.Error(entries)
	case capnslog.WARNING, capnslog.NOTICE:
		l.Warning(entries)
	case capnslog.INFO:
		l.Info(entries)
	case capnslog.DEBUG:
		fallthrough
	case capnslog.TRACE:
		l.Debug(entries)
	default:
		l.Warning("Unknown log level", level)
	}
}

func (formatter *capnslogToLogrusLogFormatter) Flush() {
}

func initEtcdLogger(conf *EtcdServerConf) (err error) {

	etcdLogger, err := capnslog.GetRepoLogger("github.com/coreos/etcd")
	if err != nil {
		return
	}

	logLevel, err := capnslog.ParseLevel(strings.ToUpper(conf.LogLevel))
	if err != nil {
		return
	}

	etcdLogger.SetRepoLogLevel(logLevel)
	capnslog.SetFormatter(&capnslogToLogrusLogFormatter{})

	return
}
