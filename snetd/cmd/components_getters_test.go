package cmd

import (
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/etcddb"
	"github.com/singnet/snet-daemon/v6/pricing"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentsCachedGetters(t *testing.T) {
	etcdClient := &etcddb.EtcdClient{}
	etcdServer := &etcddb.EtcdServer{}
	priceStrategy := &pricing.PricingStrategy{}
	modelStorage := &training.ModelStorage{}
	modelUserStorage := &training.ModelUserStorage{}
	pendingModelStorage := &training.PendingModelStorage{}
	publicModelStorage := &training.PublicModelStorage{}
	processor := blockchain.NewMockProcessor(true)

	components := &Components{
		etcdClient:          etcdClient,
		etcdServer:          etcdServer,
		priceStrategy:       priceStrategy,
		modelStorage:        modelStorage,
		modelUserStorage:    modelUserStorage,
		pendingModelStorage: pendingModelStorage,
		publicModelStorage:  publicModelStorage,
		blockchain:          processor,
	}

	assert.Same(t, etcdClient, components.EtcdClient())
	assert.Same(t, etcdServer, components.EtcdServer())
	assert.Same(t, priceStrategy, components.PricingStrategy())
	assert.Same(t, modelStorage, components.ModelStorage())
	assert.Same(t, modelUserStorage, components.ModelUserStorage())
	assert.Same(t, pendingModelStorage, components.PendingModelStorage())
	assert.Same(t, publicModelStorage, components.PublicModelStorage())
	assert.Same(t, processor, components.Blockchain())
}

func TestComponentsEtcdClientFailsToConnect(t *testing.T) {
	originalGroupName := config.GetString(config.DaemonGroupName)
	config.Vip().Set(config.DaemonGroupName, "default_group")
	t.Cleanup(func() {
		config.Vip().Set(config.DaemonGroupName, originalGroupName)
	})

	metaData, err := blockchain.InitOrganizationMetaDataFromJson([]byte(`{
		"org_name": "test",
		"org_id": "test-org",
		"groups": [{
			"group_name": "default_group",
			"group_id": "99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=",
			"payment": {
				"payment_address": "0x671276c61943A35D5F230d076bDFd91B0c47bF09",
				"payment_channel_storage_client": {
					"connection_timeout": "100ms",
					"request_timeout": "100ms",
					"endpoints": ["http://127.0.0.1:1"]
				}
			}
		}]
	}`))
	require.NoError(t, err)

	components := &Components{organizationMetaData: metaData}

	require.Panics(t, func() {
		_ = components.EtcdClient()
	})
}
