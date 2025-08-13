//go:generate go run ../resources/generate-smart-binds/main.go
package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

type Processor interface {
	ReconnectToWsClient() error
	ConnectToWsClient() error
	Enabled() bool
	EscrowContractAddress() common.Address
	MultiPartyEscrow() *MultiPartyEscrow
	GetEthHttpClient() *ethclient.Client
	GetEthWSClient() *ethclient.Client
	CurrentBlock() (*big.Int, error)
	CompareWithLatestBlockNumber(blockNumberPassed *big.Int, allowedBlockChainDifference uint64) error
	HasIdentity() bool
	Close()
	MultiPartyEscrowChannel(channelID *big.Int) (channel *MultiPartyEscrowChannel, ok bool, err error)
}

var (
	// HashPrefix32Bytes is an Ethereum signature prefix: see https://github.com/ethereum/go-ethereum/blob/bf468a81ec261745b25206b2a596eb0ee0a24a74/internal/ethapi/api.go#L361
	HashPrefix32Bytes = []byte("\x19Ethereum Signed Message:\n32")
	hashPrefix42Bytes = []byte("\x19Ethereum Signed Message:\n420x")
)

type jobInfo struct {
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

type processor struct {
	enabled                 bool
	ethHttpClient           *ethclient.Client
	rawHttpClient           *rpc.Client
	ethWSClient             *ethclient.Client
	rawWSClient             *rpc.Client
	sigHasher               func([]byte) []byte
	privateKey              *ecdsa.PrivateKey // deprecated
	address                 string
	jobCompletionQueue      chan *jobInfo
	escrowContractAddress   common.Address
	registryContractAddress common.Address
	multiPartyEscrow        *MultiPartyEscrow
}

// NewProcessor creates a new blockchain processor
func NewProcessor(metadata *ServiceMetadata) (Processor, error) {
	// TODO(aiden) accept configuration as a parameter

	p := processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
		enabled:            config.GetBool(config.BlockchainEnabledKey),
	}

	if !p.enabled {
		return &p, nil
	}

	// Setup ethereum client
	if ethHttpClients, err := CreateHTTPEthereumClient(); err != nil {
		return &p, errors.Wrap(err, "error creating RPC client")
	} else {
		p.rawHttpClient = ethHttpClients.RawClient
		p.ethHttpClient = ethHttpClients.EthClient
	}

	// TODO: if address is not in config, try to load it using network
	// TODO: Read this from github

	p.escrowContractAddress = metadata.GetMpeAddress()

	if mpe, err := NewMultiPartyEscrow(p.escrowContractAddress, p.ethHttpClient); err != nil {
		return &p, errors.Wrap(err, "error instantiating MultiPartyEscrow contract")
	} else {
		p.multiPartyEscrow = mpe
	}

	// set a local signature hash creator
	p.sigHasher = func(i []byte) []byte {
		return crypto.Keccak256(HashPrefix32Bytes, crypto.Keccak256(i))
	}

	return &p, nil
}

func (processor *processor) ReconnectToWsClient() error {
	processor.ethWSClient.Close()
	processor.rawHttpClient.Close()

	zap.L().Debug("Try to reconnect to websocket client")

	return processor.ConnectToWsClient()
}

func (processor *processor) ConnectToWsClient() error {

	zap.L().Debug("Try to connect to websocket client")

	newEthWSClients, err := CreateWSEthereumClient()
	if err != nil {
		return err
	}

	processor.ethWSClient = newEthWSClients.EthClient
	processor.rawWSClient = newEthWSClients.RawClient

	return nil
}

func (processor *processor) Enabled() (enabled bool) {
	return processor.enabled
}

func (processor *processor) EscrowContractAddress() common.Address {
	return processor.escrowContractAddress
}

func (processor *processor) MultiPartyEscrow() *MultiPartyEscrow {
	return processor.multiPartyEscrow
}

func (processor *processor) GetEthHttpClient() *ethclient.Client {
	return processor.ethHttpClient
}

func (processor *processor) GetEthWSClient() *ethclient.Client {
	return processor.ethWSClient
}

func (processor *processor) CurrentBlock() (currentBlock *big.Int, err error) {
	latestBlock, err := processor.ethHttpClient.BlockNumber(context.Background())
	if err != nil {
		zap.L().Error("error determining current block", zap.Error(err))
		return nil, fmt.Errorf("error determining current block: %v", err)
	}
	return new(big.Int).SetUint64(latestBlock), nil
}

func (processor *processor) CompareWithLatestBlockNumber(blockNumberPassed *big.Int, allowedBlockChainDifference uint64) (err error) {
	latestBlockNumber, err := processor.CurrentBlock()
	if err != nil {
		return err
	}

	differenceInBlockNumber := blockNumberPassed.Sub(blockNumberPassed, latestBlockNumber)
	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > allowedBlockChainDifference {
		return fmt.Errorf("authentication failed as the signature passed has expired")
	}
	return
}

func (processor *processor) HasIdentity() bool {
	return processor.address != ""
}

func (processor *processor) Close() {
	processor.ethHttpClient.Close()
	processor.rawHttpClient.Close()
	if processor.ethWSClient != nil {
		processor.ethWSClient.Close()
	}
	if processor.rawWSClient != nil {
		processor.rawWSClient.Close()
	}
}
