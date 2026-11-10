package contractlistener

import (
	"context"
	"slices"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/etcddb"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"go.uber.org/zap"
)

// Hooks are package level variables so that it is possible to
// mock them in the tests
var (
	getOrganizationMetaData        = blockchain.GetOrganizationMetaData
	getOrganizationMetaDataContext = blockchain.GetOrganizationMetaDataContext
	reconnectEtcd                  = etcddb.Reconnect
)

func (l *ContractEventListener) ListenOrganizationMetadataChanging() {
	l.ListenOrganizationMetadataChangingContext(context.Background())
}

func (l *ContractEventListener) ListenOrganizationMetadataChangingContext(ctx context.Context) {
	originalClient := l.CurrentEtcdClient
	defer func() {
		if l.CurrentEtcdClient != nil && l.CurrentEtcdClient != originalClient {
			l.CurrentEtcdClient.Close()
		}
	}()
	for ctx.Err() == nil {
		l.watchOrganization(ctx)
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
		if connector, ok := l.BlockchainProcessor.(interface{ ConnectToWsClientContext(context.Context) error }); ok {
			if client := l.BlockchainProcessor.GetEthWSClient(); client != nil {
				client.Close()
			}
			if err := connector.ConnectToWsClientContext(ctx); err != nil && ctx.Err() == nil {
				zap.L().Warn("websocket reconnect failed", zap.Error(err))
			}
		} else {
			_ = l.BlockchainProcessor.ReconnectToWsClient()
		}
	}
}

func (l *ContractEventListener) watchOrganization(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	zap.L().Debug("Starting contract event listener for organization metadata changing")

	watchOpts := &bind.WatchOpts{
		Start:   nil,
		Context: ctx,
	}

	ethWSClient := l.BlockchainProcessor.GetEthWSClient()
	if ethWSClient == nil {
		var err error
		if connector, ok := l.BlockchainProcessor.(interface{ ConnectToWsClientContext(context.Context) error }); ok {
			err = connector.ConnectToWsClientContext(ctx)
		} else {
			err = l.BlockchainProcessor.ConnectToWsClient()
		}
		if err != nil {
			zap.L().Warn("[ListenOrganizationMetadataChanging]", zap.Error(err))
			return
		}
		ethWSClient = l.BlockchainProcessor.GetEthWSClient()
	}
	if ethWSClient == nil {
		return
	}

	registryFilterer := blockchain.GetRegistryFilterer(ethWSClient)
	orgIdFilter := utils.MakeTopicFilterer(l.CurrentOrganizationMetaData.OrgID)

	eventContractChannel := make(chan *blockchain.RegistryOrganizationModified)
	sub, err := registryFilterer.WatchOrganizationModified(watchOpts, eventContractChannel, orgIdFilter)

	if err != nil {
		zap.L().Error("Failed to subscribe to logs", zap.Error(err))
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-sub.Err():
			if !ok {
				return
			}
			if err != nil {
				zap.L().Warn("Subscription error: ", zap.Error(err))
			}
			return
		case logData, ok := <-eventContractChannel:
			if !ok {
				return
			}
			zap.L().Debug("Log received", zap.Any("value", logData))
			metadata, err := getOrganizationMetaDataContext(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				zap.L().Warn("unable to reload organization metadata", zap.Error(err))
				continue
			}
			l.applyOrganizationMetadata(metadata)
		}
	}
}

// handleOrganizationModifiedEvent is called when the organization metadata is
// changed on blockchain. It fetches the new metadata and, if payment storage
// endpoints changed, re-establishes the etcd connection.
func (l *ContractEventListener) handleOrganizationModifiedEvent() {
	// Get metaDataUri from smart contract and organizationMetaData from IPFS
	newOrganizationMetaData := getOrganizationMetaData()
	l.applyOrganizationMetadata(newOrganizationMetaData)
}

func (l *ContractEventListener) applyOrganizationMetadata(newOrganizationMetaData *blockchain.OrganizationMetaData) {
	zap.L().Info("Get new organization metadata", zap.Any("value", newOrganizationMetaData))

	if slices.Compare(l.CurrentOrganizationMetaData.GetPaymentStorageEndPoints(), newOrganizationMetaData.GetPaymentStorageEndPoints()) != 0 {
		l.CurrentEtcdClient.Close()
		newEtcdbClient, err := reconnectEtcd(newOrganizationMetaData)
		if err != nil {
			zap.L().Error("Error in reconnecting to etcd", zap.Error(err))
		}
		l.CurrentEtcdClient = newEtcdbClient
	}

	l.CurrentOrganizationMetaData = newOrganizationMetaData
	zap.L().Info("Update current organization metadata", zap.Any("value", l.CurrentOrganizationMetaData))
}
