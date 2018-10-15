package etcddb

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/viper"
	"go.etcd.io/etcd/embed"
)

// EtcdServer struct has some useful methods to wolrk with etcd server
type EtcdServer struct {
	etcd *embed.Etcd
}

// InitEtcdServer run etcd server according to the config
// reuturns null if PAYMENT_CHANNEL_STORAGE property is not defined
// in the config file or the ENABLED field of the PAYMENT_CHANNEL_STORAGE
// is set to false
func InitEtcdServer(vip *viper.Viper) (server *EtcdServer, err error) {

	cluster := GetPaymentChannelCluster(vip)

	conf, err := GetPaymentChannelStorageServerConf(vip)

	if err != nil || conf == nil || !conf.Enabled {
		return
	}

	etcd, err := startEtcdServer(conf, cluster)
	server = &EtcdServer{etcd: etcd}
	return
}

// Close closes etcd server
func (server *EtcdServer) Close() {
	server.etcd.Close()
}

// StartEtcdServer starts ectd server
// The method blocks until the server is started
// or failed by timeout
func startEtcdServer(
	conf *PaymentChannelStorageServerConf,
	cluster string,
) (etcd *embed.Etcd, err error) {

	etcdConf := getEtcdConf(conf, cluster)
	etcd, err = embed.StartEtcd(etcdConf)

	if err != nil {
		return
	}

	select {
	case <-etcd.Server.ReadyNotify():
	case <-time.After(60 * time.Second):
		etcd.Server.Stop()
		return nil, errors.New("etcd server took too long to start: " + conf.ID)
	}

	return
}

func getEtcdConf(conf *PaymentChannelStorageServerConf, cluster string) *embed.Config {

	clientURL := &url.URL{
		Scheme: conf.Scheme,
		Host:   fmt.Sprintf("%s:%d", conf.Host, conf.ClientPort),
	}

	peerURL := &url.URL{
		Scheme: conf.Scheme,
		Host:   fmt.Sprintf("%s:%d", conf.Host, conf.PeerPort),
	}

	fmt.Printf("name: '%s'\n", conf.ID)
	fmt.Printf("client url: '%v'\n", clientURL)
	fmt.Printf("peer   url: '%v'\n", peerURL)
	fmt.Printf("initial cluster: '%s'\n", cluster)

	etcdConf := embed.NewConfig()
	etcdConf.Name = conf.ID
	etcdConf.Dir = conf.ID + ".etcd"

	// --listen-client-urls
	etcdConf.LCUrls = []url.URL{*clientURL}

	// --advertise-client-urls
	etcdConf.ACUrls = []url.URL{*clientURL}

	// --listen-peer-urls
	etcdConf.LPUrls = []url.URL{*peerURL}

	// --initial-advertise-peer-urls
	etcdConf.APUrls = []url.URL{*peerURL}

	// --initial-cluster
	etcdConf.InitialCluster = cluster

	//--initial-cluster-token
	etcdConf.InitialClusterToken = conf.Token

	//  --initial-cluster-state
	etcdConf.ClusterState = embed.ClusterStateFlagNew

	return etcdConf
}
