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

// Processor exposes Ethereum RPC clients, block freshness checks, and reads of
// MPE channel state. The daemon prepares payment claims separately; creating this
// processor does not submit transactions or establish a channel-signing identity.
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
	// HashPrefix32Bytes is the Ethereum signed-message prefix for a 32-byte digest,
	// matching the convention used by utils.GetSignature and MPE claim verification.
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

// NewProcessor creates an HTTP RPC client and an MPE binding at the address in
// service metadata when blockchain integration is enabled. Disabled mode creates
// no clients or bindings; WebSocket connectivity is initialized separately.
func NewProcessor(metadata *ServiceMetadata) (Processor, error) {
	// TODO(aiden) accept configuration as a parameter

	p := processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
		enabled:            config.GetBool(config.BlockchainEnabledKey),
	}

	if !p.enabled {
		return &p, nil
	}

	// HTTP RPC configuration is independent of the MPE address in service metadata.
	if ethHttpClients, err := CreateHTTPEthereumClient(); err != nil {
		return &p, errors.Wrap(err, "error creating RPC client")
	} else {
		p.rawHttpClient = ethHttpClients.RawClient
		p.ethHttpClient = ethHttpClients.EthClient
	}

	// The MPE address comes from service metadata, not a Registry contract field
	// or a direct per-network fallback in this constructor.

	p.escrowContractAddress = metadata.GetMpeAddress()

	if mpe, err := NewMultiPartyEscrow(p.escrowContractAddress, p.ethHttpClient); err != nil {
		return &p, errors.Wrap(err, "error instantiating MultiPartyEscrow contract")
	} else {
		p.multiPartyEscrow = mpe
	}

	// Hash the payload before applying the Ethereum 32-byte signed-message prefix.
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

// CurrentBlock reads eth_blockNumber over HTTP for each call; it does not return
// a cached block height or start a polling loop.
func (processor *processor) CurrentBlock() (currentBlock *big.Int, err error) {
	latestBlock, err := processor.ethHttpClient.BlockNumber(context.Background())
	if err != nil {
		zap.L().Error("error determining current block", zap.Error(err))
		return nil, fmt.Errorf("error determining current block: %v", err)
	}
	return new(big.Int).SetUint64(latestBlock), nil
}

// CompareWithLatestBlockNumber checks the caller-supplied inclusive window in
// either direction. The allowance is protocol-specific, not fixed by MPE/Registry.
func (processor *processor) CompareWithLatestBlockNumber(blockNumberPassed *big.Int, allowedBlockChainDifference uint64) (err error) {
	return NewEthereumBlockValidator(processor.CurrentBlock).ValidateBlock(blockNumberPassed, allowedBlockChainDifference)
}

// HasIdentity reports whether the legacy address field is populated.
// NewProcessor does not initialize a signing identity from daemon keys.
func (processor *processor) HasIdentity() bool {
	return processor.address != ""
}

func (processor *processor) Close() {
	if processor.ethHttpClient != nil {
		processor.ethHttpClient.Close()
	}
	if processor.rawHttpClient != nil {
		processor.rawHttpClient.Close()
	}
	if processor.ethWSClient != nil {
		processor.ethWSClient.Close()
	}
	if processor.rawWSClient != nil {
		processor.rawWSClient.Close()
	}
}
