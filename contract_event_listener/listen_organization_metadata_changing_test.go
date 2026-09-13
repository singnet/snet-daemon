package contractlistener

import (
	"errors"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/etcddb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	orgGroupSameEndpoints = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"

	orgGroupOtherEndpoints = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",        \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2479\",             \"http://127.0.0.1:2579\"           ]         }       }     }   ] }"
)

func orgMetadataFromJSON(t *testing.T, json string) *blockchain.OrganizationMetaData {
	t.Helper()
	metadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(json))
	require.NoError(t, err)
	require.NotNil(t, metadata)
	return metadata
}

// etcdClientSpy allows to observe Close() calls on the current etcd client
type etcdClientSpy struct {
	*etcddb.EtcdClient
	onClose func()
}

func (s *etcdClientSpy) Close() {
	if s.onClose != nil {
		s.onClose()
	}
	s.EtcdClient.Close()
}

// stubHooks replaces package level hooks for the duration of the test
func stubHooks(
	t *testing.T,
	newMetadata *blockchain.OrganizationMetaData,
	reconnect func(*blockchain.OrganizationMetaData) (*etcddb.EtcdClient, error),
) {
	t.Helper()

	oldMetadata := getOrganizationMetaData
	oldReconnect := reconnectEtcd
	t.Cleanup(func() {
		getOrganizationMetaData = oldMetadata
		reconnectEtcd = oldReconnect
	})

	getOrganizationMetaData = func() *blockchain.OrganizationMetaData {
		return newMetadata
	}
	reconnectEtcd = reconnect
}

func newTestListener(t *testing.T, currentJSON string) (*ContractEventListener, *bool) {
	t.Helper()

	closed := new(bool)
	spy := &etcdClientSpy{
		EtcdClient: &etcddb.EtcdClient{},
		onClose:    func() { *closed = true },
	}

	return &ContractEventListener{
		CurrentOrganizationMetaData: orgMetadataFromJSON(t, orgGroupSameEndpoints),
		CurrentEtcdClient:           spy,
	}, closed
}

func TestHandleOrganizationModifiedEvent(t *testing.T) {
	t.Run("same endpoints: metadata updated, etcd not reconnected", func(t *testing.T) {
		sameMetadata := orgMetadataFromJSON(t, orgGroupSameEndpoints)
		stubHooks(t, sameMetadata, nil)
		reconnectEtcd = func(*blockchain.OrganizationMetaData) (*etcddb.EtcdClient, error) {
			t.Error("reconnect should not be called when endpoints are unchanged")
			return nil, nil
		}

		listener, closed := newTestListener(t, orgGroupSameEndpoints)
		listener.handleOrganizationModifiedEvent()

		assert.False(t, *closed, "etcd client should not be closed when endpoints are unchanged")
		assert.Equal(t, sameMetadata, listener.CurrentOrganizationMetaData)
	})

	t.Run("endpoints changed: etcd closed and reconnected", func(t *testing.T) {
		newMetadata := orgMetadataFromJSON(t, orgGroupOtherEndpoints)
		newClient := &etcddb.EtcdClient{}
		stubHooks(t, newMetadata, func(*blockchain.OrganizationMetaData) (*etcddb.EtcdClient, error) {
			return newClient, nil
		})

		listener, closed := newTestListener(t, orgGroupSameEndpoints)
		listener.handleOrganizationModifiedEvent()

		assert.True(t, *closed, "old etcd client should be closed when endpoints changed")
		assert.Equal(t, newClient, listener.CurrentEtcdClient, "etcd client should be replaced with the new one")
		assert.Equal(t, newMetadata, listener.CurrentOrganizationMetaData)
	})

	t.Run("reconnect error leaves nil client", func(t *testing.T) {
		newMetadata := orgMetadataFromJSON(t, orgGroupOtherEndpoints)
		stubHooks(t, newMetadata, func(*blockchain.OrganizationMetaData) (*etcddb.EtcdClient, error) {
			return nil, errors.New("reconnect failed")
		})

		listener, closed := newTestListener(t, orgGroupSameEndpoints)
		listener.handleOrganizationModifiedEvent()

		assert.True(t, *closed, "old etcd client should be closed even if reconnect fails")
		assert.Nil(t, listener.CurrentEtcdClient)
		assert.Equal(t, newMetadata, listener.CurrentOrganizationMetaData)
	})
}
