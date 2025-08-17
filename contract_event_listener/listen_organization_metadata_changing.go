package contractlistener

import (
	"context"
	"slices"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/etcddb"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func (l *ContractEventListener) ListenOrganizationMetadataChanging() {
	zap.L().Debug("Starting contract event listener for organization metadata changing")

	watchOpts := &bind.WatchOpts{
		Start:   nil,
		Context: context.Background(),
	}

	ethWSClient := l.BlockchainProcessor.GetEthWSClient()
	if ethWSClient == nil {
		err := l.BlockchainProcessor.ConnectToWsClient()
		if err != nil {
			zap.L().Warn("[ListenOrganizationMetadataChanging]", zap.Error(err))
		}
	}

	registryFilterer := blockchain.GetRegistryFilterer(ethWSClient)
	orgIdFilter := utils.MakeTopicFilterer(l.CurrentOrganizationMetaData.OrgID)

	eventContractChannel := make(chan *blockchain.RegistryOrganizationModified)
	sub, err := registryFilterer.WatchOrganizationModified(watchOpts, eventContractChannel, orgIdFilter)

	if err != nil {
		zap.L().Error("Failed to subscribe to logs", zap.Error(err))
	}

	for {
		select {
		case err := <-sub.Err():
			if err != nil {
				zap.L().Warn("Subscription error: ", zap.Error(err))
				if websocket.IsCloseError(
					err,
					websocket.CloseNormalClosure,
					websocket.CloseAbnormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseServiceRestart,
					websocket.CloseTryAgainLater,
					websocket.CloseTLSHandshake,
				) {
					err = l.BlockchainProcessor.ReconnectToWsClient()
					if err != nil {
						zap.L().Error("Error in reconnecting to websockets", zap.Error(err))
					}
				}
			}
		case logData := <-eventContractChannel:
			zap.L().Debug("Log received", zap.Any("value", logData))

			// Get metaDataUri from smart contract and organizationMetaData from IPFS
			newOrganizationMetaData := blockchain.GetOrganizationMetaData()
			zap.L().Info("Get new organization metadata", zap.Any("value", newOrganizationMetaData))

			if slices.Compare(l.CurrentOrganizationMetaData.GetPaymentStorageEndPoints(), newOrganizationMetaData.GetPaymentStorageEndPoints()) != 0 {
				l.CurrentEtcdClient.Close()
				newEtcdbClient, err := etcddb.Reconnect(newOrganizationMetaData)
				if err != nil {
					zap.L().Error("Error in reconnecting to etcd", zap.Error(err))
				}
				l.CurrentEtcdClient = newEtcdbClient
			}

			l.CurrentOrganizationMetaData = newOrganizationMetaData
			zap.L().Info("Update current organization metadata", zap.Any("value", l.CurrentOrganizationMetaData))
		}
	}
}
