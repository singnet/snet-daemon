package escrow

import (
	"bytes"
	"net"
	"testing"
	"text/template"

	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/etcddb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var etcdPaymentHandler escrowPaymentHandler
var cleanableEtcdStorage cleanableEtcdStorageType

type cleanableEtcdStorageType struct {
	*etcddb.EtcdClient
	keys []string
}

func (storage *cleanableEtcdStorageType) Put(key string, state string) (err error) {
	storage.keys = append(storage.keys, key)
	return storage.EtcdClient.Put(key, state)
}

func (storage *cleanableEtcdStorageType) CompareAndSwap(
	key string,
	prevState string,
	newState string,
) (ok bool, err error) {
	storage.keys = append(storage.keys, key)
	return storage.EtcdClient.CompareAndSwap(key, prevState, newState)
}

func (storage *cleanableEtcdStorageType) Clear() {
	for _, key := range storage.keys {
		storage.EtcdClient.Delete(key)
	}
	storage.keys = nil
}

type EtcdStorageTemplateType struct {
	ClientPort int
	PeerPort   int
}

func initEtcdStorage() (close func(), err error) {

	confJSON, err := getEtcdJSONConf()
	if err != nil {
		return
	}

	vip := viper.New()
	err = config.ReadConfigFromJsonString(vip, confJSON)
	if err != nil {
		return
	}

	server, err := etcddb.GetEtcdServerFromVip(vip)
	if err != nil {
		return
	}

	err = server.Start()
	if err != nil {
		return
	}

	client, err := etcddb.NewEtcdClientFromVip(vip)
	if err != nil {
		return
	}

	cleanableEtcdStorage = cleanableEtcdStorageType{EtcdClient: client}

	return func() {
		server.Close()
		client.Close()
	}, nil
}

func getEtcdJSONConf() (json string, err error) {
	const confJSONTemplate = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLIENT": {
			"CONNECTION_TIMEOUT": 5000,
			"REQUEST_TIMEOUT": 3000,
			"ENDPOINTS": ["http://127.0.0.1:{{.ClientPort}}"]
		},
		"PAYMENT_CHANNEL_STORAGE_SERVER": {
			"ID": "storage-1",
			"HOST" : "127.0.0.1",
			"CLIENT_PORT": {{.ClientPort}},
			"PEER_PORT": {{.PeerPort}},
			"TOKEN": "unique-token",
			"CLUSTER": "storage-1=http://127.0.0.1:{{.PeerPort}}",
			"ENABLED": true
		}
	}`

	tmpl, err := template.New("etcd").Parse(confJSONTemplate)
	if err != nil {
		return
	}

	clientPort, err := getFreePort()
	if err != nil {
		return
	}

	peerPort, err := getFreePort()
	if err != nil {
		return
	}

	data := EtcdStorageTemplateType{
		ClientPort: clientPort,
		PeerPort:   peerPort,
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, data)
	if err != nil {
		return
	}

	json = buff.String()

	log.WithFields(log.Fields{
		"ClientPort": clientPort,
		"PeerPort":   peerPort,
	}).Info()

	log.Info("EtcdConfig", json)

	return
}

func getFreePort() (port int, err error) {

	listener, err := net.Listen("tcp", ":0")

	if err != nil {
		return
	}

	defer listener.Close()

	port = listener.Addr().(*net.TCPAddr).Port
	return
}

func TestEtcdGetPayment(t *testing.T) {

	close, e := initEtcdStorage()
	if e != nil {
		t.Errorf("error during etcd storage initialization: %v", e)
	}
	defer close()

	err := cleanableEtcdStorage.Put("key", "value")
	assert.Nil(t, err)

	value, ok, err := cleanableEtcdStorage.Get("key")
	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, "value", value)
}
