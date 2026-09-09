package blockchain

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/websocket"
	"github.com/singnet/snet-daemon/v6/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const metadataJson = "{\n    \"version\": 1,\n    \"display_name\": \"semyon_dev\",\n    \"encoding\": \"proto\",\n    \"service_type\": \"grpc\",\n    \"service_api_source\": \"ipfs://QmV9bBsLAZfXGibdU3isPwDn8SxAPkqg6YzcKamSCxnBCR\",\n    \"mpe_address\": \"0x7E0aF8988DF45B824b2E0e0A87c6196897744970\",\n    \"groups\": [\n        {\n            \"group_name\": \"default_group\",\n            \"endpoints\": [\n                \"http://localhost:7000\"\n            ],\n            \"pricing\": [\n                {\n                    \"price_model\": \"fixed_price\",\n                    \"price_in_cogs\": 1,\n                    \"default\": true\n                }\n            ],\n            \"group_id\": \"FtNuizEOUsVCd5f2Fij9soehtRSb58LlTePgkVnsgVI=\",\n            \"free_call_signer_address\": \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\",\n            \"free_calls\": 25,\n            \"daemon_addresses\": [\n                \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\"\n            ]\n        },\n        {\n            \"group_name\": \"not_default\",\n            \"endpoints\": [\n                \"http://localhost:7000\"\n            ],\n            \"pricing\": [\n                {\n                    \"price_model\": \"fixed_price\",\n                    \"price_in_cogs\": 1,\n                    \"default\": true\n                }\n            ],\n            \"group_id\": \"udN0SLIvsDdvQQe3Ltv/NwqCh7sPKdz4scYmlI7AMdE=\",\n            \"free_call_signer_address\": \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\",\n            \"free_calls\": 35,\n            \"daemon_addresses\": [\n                \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\"\n            ]\n        }\n    ],\n    \"assets\": {},\n    \"media\": [],\n    \"tags\": [],\n    \"service_description\": {\n        \"description\": \"Test service with localhost endpoint!\",\n        \"url\": \"\"\n    }\n}"

// ProcessorTestSuite is a test suite for the processor struct
type ProcessorTestSuite struct {
	suite.Suite
	processor Processor
}

// SetupSuite initializes the Ethereum client before running the tests
func (suite *ProcessorTestSuite) SetupSuite() {
	config.Vip().Set(config.BlockchainEnabledKey, true)
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	_, err := InitServiceMetaDataFromJson([]byte(metadataJson))
	assert.Nil(suite.T(), err)
	suite.processor = NewMockProcessor(true)
}

// Test: If the block number difference is within the allowed limit → no error
func (suite *ProcessorTestSuite) TestCompareWithLatestBlockNumber_WithinLimit() {
	latestBlock, err := suite.processor.CurrentBlock()
	suite.Require().NoError(err, "CurrentBlock() should not return an error")

	// Simulate a block number within the allowed range (+2)
	blockNumberPassed := new(big.Int).Add(latestBlock, big.NewInt(2))
	err = suite.processor.CompareWithLatestBlockNumber(blockNumberPassed, 5)

	// Expect no error
	assert.NoError(suite.T(), err, "Expected no error when block difference is within the limit")
}

// Test: If the block number difference exceeds the allowed limit → return an error
func (suite *ProcessorTestSuite) TestCompareWithLatestBlockNumber_ExceedsLimit() {
	latestBlock, err := suite.processor.CurrentBlock()
	suite.Require().NoError(err, "CurrentBlock() should not return an error")

	// Simulate a block number exceeding the allowed limit (+10)
	blockNumberPassed := new(big.Int).Add(latestBlock, big.NewInt(10))
	err = suite.processor.CompareWithLatestBlockNumber(blockNumberPassed, 5)

	// Expect an error
	assert.Error(suite.T(), err, "Expected an error when block difference exceeds the limit")
	assert.Contains(suite.T(), err.Error(), "authentication failed", "Error message should indicate signature expiration")
}

// Run the test suite
func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}

func TestNewProcessor_BlockchainDisabled(t *testing.T) {
	config.Vip().Set(config.BlockchainEnabledKey, false)
	defer config.Vip().Set(config.BlockchainEnabledKey, true)

	processor, err := NewProcessor(&ServiceMetadata{})

	assert.NoError(t, err)
	assert.NotNil(t, processor)
	assert.False(t, processor.Enabled())
	assert.False(t, processor.HasIdentity())
	assert.Nil(t, processor.GetEthHttpClient())
	assert.Nil(t, processor.GetEthWSClient())
	assert.Nil(t, processor.MultiPartyEscrow())
}

func TestProcessorGetters(t *testing.T) {
	escrowAddress := common.HexToAddress("0x7E0aF8988DF45B824b2E0e0A87c6196894970")
	p := &processor{
		enabled:               true,
		address:               "0x747155e03c892B8b311B7Cfbb920664E8c6792fA",
		escrowContractAddress: escrowAddress,
		multiPartyEscrow:      &MultiPartyEscrow{},
	}

	assert.True(t, p.HasIdentity())
	assert.True(t, p.Enabled())
	assert.Equal(t, escrowAddress, p.EscrowContractAddress())
	assert.NotNil(t, p.MultiPartyEscrow())
	assert.Nil(t, p.GetEthHttpClient())
	assert.Nil(t, p.GetEthWSClient())

	p.address = ""
	assert.False(t, p.HasIdentity())

	p.enabled = false
	assert.False(t, p.Enabled())
}

func TestProcessorHasIdentityEmpty(t *testing.T) {
	p := &processor{}

	assert.False(t, p.HasIdentity())
	assert.False(t, p.Enabled())
	assert.Equal(t, common.Address{}, p.EscrowContractAddress())
	assert.Nil(t, p.MultiPartyEscrow())
}

// startMockEthRPCServer starts a local JSON-RPC server responding to
// eth_blockNumber requests with the given hex result
func startMockEthRPCServer(t *testing.T, blockNumberResult string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  blockNumberResult,
		})
	}))
	return server
}

func setBlockChainTestEndpoint(t *testing.T, endpoint string) {
	t.Helper()
	originalEndpoint := config.GetString(config.EthereumJsonRpcHTTPEndpointKey)
	config.Vip().Set(config.EthereumJsonRpcHTTPEndpointKey, endpoint)
	assert.NoError(t, config.Validate())

	t.Cleanup(func() {
		config.Vip().Set(config.EthereumJsonRpcHTTPEndpointKey, originalEndpoint)
		assert.NoError(t, config.Validate())
	})
}

func TestNewProcessorWithMockRPCServer(t *testing.T) {
	server := startMockEthRPCServer(t, "0x64")
	defer server.Close()
	setBlockChainTestEndpoint(t, server.URL)

	metaData := &ServiceMetadata{multiPartyEscrowAddress: common.HexToAddress("0x7E0aF8988DF45B824b2E0e0A87c6196894970")}

	processor, err := NewProcessor(metaData)

	assert.NoError(t, err)
	assert.NotNil(t, processor)
	assert.True(t, processor.Enabled())
	assert.NotNil(t, processor.GetEthHttpClient())
	assert.Nil(t, processor.GetEthWSClient())
	assert.Equal(t, metaData.GetMpeAddress(), processor.EscrowContractAddress())
	assert.NotNil(t, processor.MultiPartyEscrow())

	block, err := processor.CurrentBlock()
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100), block)

	// block number within the allowed difference
	assert.NoError(t, processor.CompareWithLatestBlockNumber(big.NewInt(102), 5))

	// block number outside the allowed difference
	err = processor.CompareWithLatestBlockNumber(big.NewInt(200), 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")

	processor.Close()
}

func TestProcessorCurrentBlockError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"error":   map[string]any{"code": -32000, "message": "server error"},
		})
	}))
	defer server.Close()
	setBlockChainTestEndpoint(t, server.URL)

	processor, err := NewProcessor(&ServiceMetadata{multiPartyEscrowAddress: common.HexToAddress("0x7E0aF8988DF45B824b2E0e0A87c6196894970")})
	assert.NoError(t, err)
	defer processor.Close()

	_, err = processor.CurrentBlock()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error determining current block")

	assert.Error(t, processor.CompareWithLatestBlockNumber(big.NewInt(100), 5))
}

func TestGetRegistryFilterer(t *testing.T) {
	server := startMockEthRPCServer(t, "0x64")
	defer server.Close()
	setBlockChainTestEndpoint(t, server.URL)

	ethClient, err := CreateHTTPEthereumClient()
	assert.NoError(t, err)
	defer ethClient.Close()

	registryFilterer := GetRegistryFilterer(ethClient.EthClient)
	assert.NotNil(t, registryFilterer)
}

func TestNewProcessor_InvalidEndpoint(t *testing.T) {
	setBlockChainTestEndpoint(t, "://invalid-endpoint")

	processor, err := NewProcessor(&ServiceMetadata{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating RPC client")
	assert.True(t, processor.Enabled())
}

func TestGetRegistryCaller_InvalidEndpoint(t *testing.T) {
	setBlockChainTestEndpoint(t, "://invalid-endpoint")

	assert.Panics(t, func() { getRegistryCaller() })
}

func setBlockChainTestEndpoints(t *testing.T, httpEndpoint, wsEndpoint string) {
	t.Helper()
	originalHttpEndpoint := config.GetString(config.EthereumJsonRpcHTTPEndpointKey)
	originalWsEndpoint := config.GetString(config.EthereumJsonRpcWSEndpointKey)
	config.Vip().Set(config.EthereumJsonRpcHTTPEndpointKey, httpEndpoint)
	config.Vip().Set(config.EthereumJsonRpcWSEndpointKey, wsEndpoint)
	assert.NoError(t, config.Validate())

	t.Cleanup(func() {
		config.Vip().Set(config.EthereumJsonRpcHTTPEndpointKey, originalHttpEndpoint)
		config.Vip().Set(config.EthereumJsonRpcWSEndpointKey, originalWsEndpoint)
		assert.NoError(t, config.Validate())
	})
}

// startMockWSServer starts a local WebSocket server which accepts connections
// and keeps them open without answering any requests
func startMockWSServer(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
}

func TestCreateWSEthereumClient_InvalidEndpoint(t *testing.T) {
	setBlockChainTestEndpoints(t, "http://127.0.0.1:1", "ws://127.0.0.1:1")

	_, err := CreateWSEthereumClient()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating RPC WebSocket client")
}

func TestConnectAndReconnectToWsClient(t *testing.T) {
	httpServer := startMockEthRPCServer(t, "0x64")
	defer httpServer.Close()
	wsServer := startMockWSServer(t)
	defer wsServer.Close()
	setBlockChainTestEndpoints(t, httpServer.URL, "ws"+strings.TrimPrefix(wsServer.URL, "http"))

	processor, err := NewProcessor(&ServiceMetadata{multiPartyEscrowAddress: common.HexToAddress("0x7E0aF8988DF45B824b2E0e0A87c6196894970")})
	assert.NoError(t, err)

	assert.NoError(t, processor.ConnectToWsClient())
	assert.NotNil(t, processor.GetEthWSClient())

	assert.NoError(t, processor.ReconnectToWsClient())

	processor.Close()
}

// encodeMockEscrowChannel ABI-encodes the result of MultiPartyEscrow.channels()
// call with fields in the order of the contract ABI outputs:
// nonce, sender, signer, recipient, groupId, value, expiration
func encodeMockEscrowChannel(nonce *big.Int, sender, signer, recipient common.Address,
	groupId [32]byte, value, expiration *big.Int) (string, error) {
	parsedAbi, err := MultiPartyEscrowMetaData.GetAbi()
	if err != nil {
		return "", err
	}
	packed, err := parsedAbi.Methods["channels"].Outputs.Pack(nonce, sender, signer, recipient, groupId, value, expiration)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(packed), nil
}

func startMockEthRPCServerWithCallResult(t *testing.T, callResult string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			ID     any    `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		result := "0x64"
		if request.Method == "eth_call" {
			result = callResult
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
}

func TestMultiPartyEscrowChannel_Found(t *testing.T) {
	sender := common.HexToAddress("0x747155e03c892B8b311B7Cfbb920664E8c6792fA")
	recipient := common.HexToAddress("0x671276c61943A35D5F230d076bDFd91B0c47bF09")
	groupId := [32]byte{1, 2, 3}

	channelResult, err := encodeMockEscrowChannel(
		big.NewInt(7), sender, sender, recipient, groupId, big.NewInt(1000), big.NewInt(555))
	assert.NoError(t, err)

	server := startMockEthRPCServerWithCallResult(t, channelResult)
	defer server.Close()
	setBlockChainTestEndpoint(t, server.URL)

	processor, err := NewProcessor(&ServiceMetadata{multiPartyEscrowAddress: common.HexToAddress("0x7E0aF8988DF45B824b2E0e0A87c6196894970")})
	assert.NoError(t, err)
	defer processor.Close()

	channel, ok, err := processor.MultiPartyEscrowChannel(big.NewInt(1))

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, sender, channel.Sender)
	assert.Equal(t, recipient, channel.Recipient)
	assert.Equal(t, groupId, channel.GroupId)
	assert.Equal(t, big.NewInt(1000), channel.Value)
	assert.Equal(t, big.NewInt(7), channel.Nonce)
	assert.Equal(t, big.NewInt(555), channel.Expiration)
}

func TestMultiPartyEscrowChannel_NotFound(t *testing.T) {
	zeroAddress := common.Address{}
	channelResult, err := encodeMockEscrowChannel(
		big.NewInt(0), zeroAddress, zeroAddress, zeroAddress, [32]byte{}, big.NewInt(0), big.NewInt(0))
	assert.NoError(t, err)

	server := startMockEthRPCServerWithCallResult(t, channelResult)
	defer server.Close()
	setBlockChainTestEndpoint(t, server.URL)

	processor, err := NewProcessor(&ServiceMetadata{multiPartyEscrowAddress: common.HexToAddress("0x7E0aF8988DF45B824b2E0e0A87c6196894970")})
	assert.NoError(t, err)
	defer processor.Close()

	channel, ok, err := processor.MultiPartyEscrowChannel(big.NewInt(1))

	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, channel)
}
