package blockchain

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/singnet/snet-daemon/v6/utils"
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

// getAuthOption returns the appropriate RPC auth option based on endpoint and apiKey.
// - Infura requires Basic Auth with an empty username and apiKey as password. Also can be replaced with jwt token.
// - Other providers (e.g. Alchemy) use Bearer token in the Authorization header.
// - If apiKey is empty, no auth header is added.
func getAuthOption(endpoint, apiKey string) rpc.ClientOption {

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}

	// infura can accept jwt
	if utils.IsJWT(apiKey) {
		return rpc.WithHeader("Authorization", "Bearer "+apiKey)
	}
	// infura need Basic auth for a classic secret key
	if strings.Contains(endpoint, "infura") {
		return rpc.WithHeader("Authorization", "Basic "+basicAuth("", apiKey))
	}

	// other ways use Bearer for most providers
	return rpc.WithHeader("Authorization", "Bearer "+apiKey)
}

func CreateHTTPEthereumClient() (*EthereumClient, error) {

	opts := getAuthOption(config.GetBlockChainHTTPEndPoint(), config.GetString(config.BlockchainProviderApiKey))

	ethereumHttpClient := new(EthereumClient)
	var httpClient *rpc.Client
	var err error
	if opts == nil {
		httpClient, err = rpc.DialOptions(
			context.Background(),
			config.GetBlockChainHTTPEndPoint())
	} else {
		httpClient, err = rpc.DialOptions(
			context.Background(),
			config.GetBlockChainHTTPEndPoint(), opts)
	}

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
