package blockchain

import (
	"context"
	"encoding/base64"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/config"
)

type EthereumClient struct {
	EthClient *ethclient.Client
	RawClient *rpc.Client
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func GetEthereumClient() (*EthereumClient, error) {

	ethereumClient := new(EthereumClient)
	if client, err := rpc.DialOptions(context.Background(),
		config.GetBlockChainEndPoint(),
		rpc.WithHeader("Authorization", "Basic "+basicAuth("", config.GetString(config.BlockchainProviderApiKey)))); err != nil {
		return nil, errors.Wrap(err, "error creating RPC client")
	} else {
		ethereumClient.RawClient = client
		ethereumClient.EthClient = ethclient.NewClient(client)
	}

	return ethereumClient, nil

}
func (ethereumClient *EthereumClient) Close() {
	if ethereumClient != nil {
		ethereumClient.EthClient.Close()
		ethereumClient.RawClient.Close()
	}
}
