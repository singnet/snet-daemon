package blockchain

import (
	"context"
	"encoding/base64"
	"go.uber.org/zap"

	"github.com/singnet/snet-daemon/v6/config"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type EthereumClient struct {
	EthClient *ethclient.Client
	RawClient *rpc.Client
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func CreateEthereumClient() (*EthereumClient, error) {
	ethereumHttpClient, err := CreateHTTPEthereumClient()
	if err != nil {
		return nil, err
	}
	return ethereumHttpClient, nil
}

func CreateHTTPEthereumClient() (*EthereumClient, error) {
	ethereumHttpClient := new(EthereumClient)
	httpClient, err := rpc.DialOptions(
		context.Background(),
		config.GetBlockChainHTTPEndPoint(),
		rpc.WithHeader("Authorization", "Basic "+basicAuth("", config.GetString(config.BlockchainProviderApiKey))))
	if err != nil {
		zap.L().Error("Error creating ethereum client", zap.Error(err), zap.String("endpoint", config.GetBlockChainHTTPEndPoint()))
		return nil, errors.Wrap(err, "error creating RPC client")
	}

	ethereumHttpClient.RawClient = httpClient
	ethereumHttpClient.EthClient = ethclient.NewClient(httpClient)
	return ethereumHttpClient, nil
}

func CreateWSEthereumClient() (*EthereumClient, error) {
	ethereumWsClient := new(EthereumClient)
	wsClient, err := rpc.DialOptions(
		context.Background(),
		config.GetBlockChainWSEndPoint(),
		rpc.WithHeader("Authorization", "Basic "+basicAuth("", config.GetString(config.BlockchainProviderApiKey))))
	if err != nil {
		return nil, errors.Wrap(err, "error creating RPC WebSocket client")
	}
	ethereumWsClient.RawClient = wsClient
	ethereumWsClient.EthClient = ethclient.NewClient(wsClient)
	return ethereumWsClient, nil
}

func (ethereumClient *EthereumClient) Close() {
	if ethereumClient != nil {
		ethereumClient.EthClient.Close()
		ethereumClient.RawClient.Close()
	}
}
