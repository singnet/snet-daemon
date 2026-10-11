package cmd

import (
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

func TestComponentsMetadataAndStorageGetters(t *testing.T) {
	originalBlockchain := config.GetBool(config.BlockchainEnabledKey)
	originalStorageType := config.GetString(config.PaymentChannelStorageTypeKey)
	config.Vip().Set(config.BlockchainEnabledKey, false)
	config.Vip().Set(config.PaymentChannelStorageTypeKey, "memory")
	t.Cleanup(func() {
		config.Vip().Set(config.BlockchainEnabledKey, originalBlockchain)
		config.Vip().Set(config.PaymentChannelStorageTypeKey, originalStorageType)
	})

	components := &Components{blockchain: blockchain.NewMockProcessor(false)}

	assert.NotNil(t, components.ServiceMetaData())
	assert.NotNil(t, components.OrganizationMetaData())
	assert.NotNil(t, components.AtomicStorage())
	assert.NotNil(t, components.MPESpecificStorage())
	assert.NotNil(t, components.FreeCallLockerStorage())
	assert.NotNil(t, components.LockerStorage())
	assert.NotNil(t, components.PaymentStorage())
	assert.NotNil(t, components.FreeCallUserStorage())
	assert.NotNil(t, components.PrepaidUserStorage())
	assert.NotNil(t, components.ChannelBroadcast())
	assert.NotNil(t, components.TokenManager())
}
