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
	"github.com/singnet/snet-daemon/v5/config"
	"go.uber.org/zap"
)

var (
	// HashPrefix32Bytes is an Ethereum signature prefix: see https://github.com/ethereum/go-ethereum/blob/bf468a81ec261745b25206b2a596eb0ee0a24a74/internal/ethapi/api.go#L361
	HashPrefix32Bytes = []byte("\x19Ethereum Signed Message:\n32")
	hashPrefix42Bytes = []byte("\x19Ethereum Signed Message:\n420x")
)

type jobInfo struct {
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

type Processor struct {
	enabled                 bool
	ethHttpClient           *ethclient.Client
	rawHttpClient           *rpc.Client
	ethWSClient             *ethclient.Client
	rawWSClient             *rpc.Client
	sigHasher               func([]byte) []byte
	privateKey              *ecdsa.PrivateKey
	address                 string
	jobCompletionQueue      chan *jobInfo
	escrowContractAddress   common.Address
	registryContractAddress common.Address
	multiPartyEscrow        *MultiPartyEscrow
}

// NewProcessor creates a new blockchain processor
func NewProcessor(metadata *ServiceMetadata) (Processor, error) {
	// TODO(aiden) accept configuration as a parameter

	p := Processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
		enabled:            config.GetBool(config.BlockchainEnabledKey),
	}

	if !p.enabled {
		return p, nil
	}

	// Setup ethereum client
	if ethHttpClients, err := CreateEthereumClient(); err != nil {
		return p, errors.Wrap(err, "error creating RPC client")
	} else {
		p.rawHttpClient = ethHttpClients.RawClient
		p.ethHttpClient = ethHttpClients.EthClient
	}

	// TODO: if address is not in config, try to load it using network
	//TODO: Read this from github

	p.escrowContractAddress = metadata.GetMpeAddress()

	if mpe, err := NewMultiPartyEscrow(p.escrowContractAddress, p.ethHttpClient); err != nil {
		return p, errors.Wrap(err, "error instantiating MultiPartyEscrow contract")
	} else {
		p.multiPartyEscrow = mpe
	}

	// set local signature hash creator
	p.sigHasher = func(i []byte) []byte {
		return crypto.Keccak256(HashPrefix32Bytes, crypto.Keccak256(i))
	}

	return p, nil
}

func (processor *Processor) ReconnectToWsClient() error {
	processor.ethWSClient.Close()
	processor.rawHttpClient.Close()

	zap.L().Debug("Try to reconnect to websocket client")

	return processor.ConnectToWsClient()
}

func (processor *Processor) ConnectToWsClient() error {

	zap.L().Debug("Try to connect to websocket client")

	newEthWSClients, err := CreateWSEthereumClient()
	if err != nil {
		return err
	}

	processor.ethWSClient = newEthWSClients.EthClient
	processor.rawWSClient = newEthWSClients.RawClient

	return nil
}

func (processor *Processor) Enabled() (enabled bool) {
	return processor.enabled
}

func (processor *Processor) EscrowContractAddress() common.Address {
	return processor.escrowContractAddress
}

func (processor *Processor) MultiPartyEscrow() *MultiPartyEscrow {
	return processor.multiPartyEscrow
}

func (processor *Processor) GetEthHttpClient() *ethclient.Client {
	return processor.ethHttpClient
}

func (processor *Processor) GetEthWSClient() *ethclient.Client {
	return processor.ethWSClient
}

func (processor *Processor) CurrentBlock() (currentBlock *big.Int, err error) {
	latestBlock, err := processor.ethHttpClient.BlockNumber(context.Background())
	if err != nil {
		zap.L().Error("error determining current block", zap.Error(err))
		return nil, fmt.Errorf("error determining current block: %v", err)
	}
	return new(big.Int).SetUint64(latestBlock), nil
}

func (processor *Processor) CompareWithLatestBlockNumber(blockNumberPassed *big.Int, allowedBlockChainDifference uint64) (err error) {
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

func (processor *Processor) HasIdentity() bool {
	return processor.address != ""
}

func (processor *Processor) Close() {
	processor.ethHttpClient.Close()
	processor.rawHttpClient.Close()
	if processor.ethWSClient != nil {
		processor.ethWSClient.Close()
	}
	if processor.rawWSClient != nil {
		processor.rawWSClient.Close()
	}
}
