package etcddb

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

// EtcdServer struct has some useful methods to work with etcd server
type EtcdServer struct {
	conf *EtcdServerConf
	etcd *embed.Etcd
}

func (server EtcdServer) GetConf() *EtcdServerConf {
	return server.conf

}

// IsEtcdServerEnabled checks that etcd server is enabled using conf file
func IsEtcdServerEnabled() (enabled bool, err error) {
	return IsEtcdServerEnabledInVip(config.Vip())
}

// IsEtcdServerEnabledInVip checks that etcd server is enabled using viper conf
func IsEtcdServerEnabledInVip(vip *viper.Viper) (enabled bool, err error) {

	conf, err := GetEtcdServerConf(vip)
	if err != nil {
		return
	}

	return conf.Enabled, nil
}

// GetEtcdServer returns EtcdServer in case it is defined in the viper config
// returns null if the PAYMENT_CHANNEL_STORAGE property is not defined
// in the config file or the ENABLED field of the PAYMENT_CHANNEL_STORAGE
// is set to false
func GetEtcdServer() (server *EtcdServer, err error) {
	return GetEtcdServerFromVip(config.Vip())
}

// GetEtcdServerFromVip run etcd server using viper config
func GetEtcdServerFromVip(vip *viper.Viper) (server *EtcdServer, err error) {

	conf, err := GetEtcdServerConf(vip)

	zap.L().Info("Getting payment channel storage sever", zap.Any("conf", conf))

	if err != nil || conf == nil || !conf.Enabled {
		return
	}

	server = &EtcdServer{conf: conf}
	return
}

// Start starts etcd server
func (server *EtcdServer) Start() (err error) {

	etcd, err := startEtcdServer(server.conf)
	if err != nil {
		return
	}

	server.etcd = etcd
	return
}

// Close closes etcd server
func (server *EtcdServer) Close() {
	server.etcd.Close()
}

// StartEtcdServer starts ectd server
// The method blocks until the server is started
// or failed by timeout
func startEtcdServer(conf *EtcdServerConf) (etcd *embed.Etcd, err error) {

	etcdConf := getEtcdConf(conf)
	etcd, err = embed.StartEtcd(etcdConf)

	if err != nil {
		return
	}

	select {
	case <-etcd.Server.ReadyNotify():
	case <-time.After(conf.StartupTimeout):
		etcd.Server.Stop()
		return nil, errors.New("etcd server took too long to start: " + conf.ID)
	}

	return
}

func getEtcdConf(conf *EtcdServerConf) *embed.Config {

	clientURL := &url.URL{
		Scheme: conf.Scheme,
		Host:   fmt.Sprintf("%s:%d", conf.Host, conf.ClientPort),
	}

	peerURL := &url.URL{
		Scheme: conf.Scheme,
		Host:   fmt.Sprintf("%s:%d", conf.Host, conf.PeerPort),
	}

	zap.L().Info("Getting etcd config", zap.Any("PaymentChannelStorageServer", fmt.Sprintf("%+v", conf)),
		zap.Any("ClientURL", clientURL),
		zap.Any("PeerURL", clientURL))

	etcdConf := embed.NewConfig()
	etcdConf.Name = conf.ID
	etcdConf.Dir = conf.DataDir

	// --listen-client-urls
	etcdConf.ListenClientUrls = []url.URL{*clientURL}

	// --advertise-client-urls
	etcdConf.AdvertiseClientUrls = []url.URL{*clientURL}

	// --listen-peer-urls
	etcdConf.ListenPeerUrls = []url.URL{*peerURL}

	// --initial-advertise-peer-urls
	etcdConf.AdvertisePeerUrls = []url.URL{*peerURL}

	// --initial-cluster
	etcdConf.InitialCluster = conf.Cluster

	//--initial-cluster-token
	etcdConf.InitialClusterToken = conf.Token

	//  --initial-cluster-state
	etcdConf.ClusterState = embed.ClusterStateFlagNew

	// --log-level
	etcdConf.LogLevel = conf.LogLevel

	// --log-outputs
	etcdConf.LogOutputs = conf.LogOutputs

	return etcdConf
}
