package escrow

import (
	"math/big"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/singnet/snet-daemon/handler"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var etcdPaymentHandler escrowPaymentHandler
var etcdStorageMock etcdStorageMockType

type etcdStorageMockType struct {
	*EtcdStorage
	keys []*PaymentChannelKey
}

func (storageMock *etcdStorageMockType) Put(key *PaymentChannelKey, state *PaymentChannelData) (err error) {
	storageMock.keys = append(storageMock.keys, key)
	return storageMock.EtcdStorage.Put(key, state)
}

func (storageMock *etcdStorageMockType) CompareAndSwap(
	key *PaymentChannelKey,
	prevState *PaymentChannelData,
	newState *PaymentChannelData,
) (ok bool, err error) {
	storageMock.keys = append(storageMock.keys, key)
	return storageMock.EtcdStorage.CompareAndSwap(key, prevState, newState)
}

func (storageMock *etcdStorageMockType) Clear() {
	for _, key := range storageMock.keys {
		bytes, _ := serialize(key)
		storageMock.client.Delete(bytes)
	}
	storageMock.keys = nil
}

func initEtcdStorage() (close func(), err error) {

	const confJSON = `
	{
		"PAYMENT_CHANNEL_STORAGE_CLUSTER": "storage-1=http://127.0.0.1:2380",
		"PAYMENT_CHANNEL_STORAGE_CLIENT": {
			"CONNECTION_TIMEOUT": 5000,
			"REQUEST_TIMEOUT": 3000
		},
		"PAYMENT_CHANNEL_STORAGE_SERVER": {
			"ID": "storage-1",
			"HOST" : "127.0.0.1",
			"CLIENT_PORT": 2379,
			"PEER_PORT": 2380,
			"TOKEN": "unique-token",
			"ENABLED": true
		}
	}`

	vip := viper.New()
	err = config.ReadConfigFromJsonString(vip, confJSON)
	if err != nil {
		return
	}

	server, err := etcddb.InitEtcdServer(vip)
	if err != nil {
		return
	}

	NewEtcdStorage(vip)

	storage, err := NewEtcdStorage(vip)

	if err != nil {
		return
	}

	etcdStorageMock = etcdStorageMockType{EtcdStorage: storage}

	etcdPaymentHandler = escrowPaymentHandler{
		escrowContractAddress: testEscrowContractAddress,
		storage:               &etcdStorageMock,
		incomeValidator:       &incomeValidatorMock,
	}

	return func() {
		server.Close()
		storage.Close()
	}, nil

}

func getTestEtcdContext(data *testPaymentData) *handler.GrpcStreamContext {
	etcdStorageMock.Put(
		newPaymentChannelKey(data.ChannelID, data.ChannelNonce),
		&PaymentChannelData{
			State:            data.State,
			Sender:           testPublicKey,
			FullAmount:       big.NewInt(data.FullAmount),
			Expiration:       data.Expiration,
			AuthorizedAmount: big.NewInt(data.PrevAmount),
			Signature:        nil,
		},
	)
	md := getEscrowMetadata(data.ChannelID, data.ChannelNonce, data.NewAmount)
	return &handler.GrpcStreamContext{
		MD: md,
	}
}

func clearTestEtcdContext() {
	etcdStorageMock.Clear()
}

func TestEtcdGetPayment(t *testing.T) {

	close, e := initEtcdStorage()
	assert.Nil(t, e)
	defer close()

	data := &testPaymentData{
		ChannelID:    42,
		ChannelNonce: 3,
		Expiration:   time.Now().Add(time.Hour),
		FullAmount:   12345,
		NewAmount:    12345,
		PrevAmount:   12300,
		State:        Open,
	}
	context := getTestEtcdContext(data)
	defer clearTestEtcdContext()

	payment, err := etcdPaymentHandler.Payment(context)

	assert.Nil(t, err)
	expected := getTestPayment(data)
	actual := payment.(*escrowPaymentType)
	assert.Equal(t, toJSON(expected.grpcContext), toJSON(actual.grpcContext))
	assert.Equal(t, toJSON(expected.channelKey), toJSON(actual.channelKey))
	assert.Equal(t, expected.amount, actual.amount)
	assert.Equal(t, expected.signature, actual.signature)
	assert.Equal(t, toJSON(expected.channel), toJSON(actual.channel))
}
