package contract_event_listener

import (
	"context"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/etcddb"
	"github.com/singnet/snet-daemon/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

func (l *ContractEventListener) ListenOrganizationMetadataChanging() {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			common.HexToAddress(contractAddress),
		},
		Topics: [][]common.Hash{
			{
				common.HexToHash(string(UpdateMetadataUriEventSignature)),
				blockchain.StringToHash(l.CurrentOrganizationMetaData.OrgID),
			},
		},
	}

	ethWSClient := l.BlockchainProcessor.GetEthWSClient()

	logs := make(chan types.Log)
	sub, err := ethWSClient.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		zap.L().Fatal("Failed to subscribe to logs", zap.Error(err))
	}

	for {
		select {
		case err := <-sub.Err():
			zap.L().Error("Subscription error: ", zap.Error(err))
		case logData := <-logs:
			zap.L().Debug("Log received", zap.Any("value", logData))

			// Get metaDataUri from smart contract and organizationMetaData from IPFS
			newOrganizationMetaData := blockchain.GetOrganizationMetaData()
			zap.L().Info("Get new organization metadata", zap.Any("value", newOrganizationMetaData))

			if !utils.CompareSlices(l.CurrentOrganizationMetaData.GetPaymentStorageEndPoints(), newOrganizationMetaData.GetPaymentStorageEndPoints()) {
				// mutex
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
