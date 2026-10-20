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

// EthereumClient exposes typed and raw views of the same RPC connection.
type EthereumClient struct {
	EthClient *ethclient.Client
	RawClient *rpc.Client
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// getAuthOption selects the daemon's HTTP RPC authentication policy. Empty keys
// add no header. JWT-shaped keys use Bearer auth; other keys use Basic auth for
// endpoints containing "infura" and Bearer auth elsewhere. IsJWT only recognizes
// token shape and does not authenticate the supplied credential.
func getAuthOption(endpoint, apiKey string) rpc.ClientOption {

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}

	// JWT-shaped credentials take precedence over the endpoint-specific rule.
	if utils.IsJWT(apiKey) {
		return rpc.WithHeader("Authorization", "Bearer "+apiKey)
	}
	// The Infura rule uses an empty username and the configured key as password.
	if strings.Contains(endpoint, "infura") {
		return rpc.WithHeader("Authorization", "Basic "+basicAuth("", apiKey))
	}

	// Other endpoints receive the supplied key as a Bearer credential.
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

// CreateWSEthereumClient uses the configured WebSocket endpoint and always adds
// Basic auth with an empty username. It does not use the HTTP auth-selection rule.
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
