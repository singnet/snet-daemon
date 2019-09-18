//go:generate nodejs ../resources/blockchain/scripts/generateAbi.js --contract-package singularitynet-platform-contracts --contract-name MultiPartyEscrow --go-package blockchain --output-file multi_party_escrow.go
//go:generate nodejs ../resources/blockchain/scripts/generateAbi.js --contract-package singularitynet-platform-contracts --contract-name Registry --go-package blockchain --output-file registry.go
package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"math/big"
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
	ethClient               *ethclient.Client
	rawClient               *rpc.Client
	sigHasher               func([]byte) []byte
	privateKey              *ecdsa.PrivateKey
	address                 string
	jobCompletionQueue      chan *jobInfo
	escrowContractAddress   common.Address
	registryContractAddress common.Address
	multiPartyEscrow        *MultiPartyEscrow
	groupId                 [32]byte
	recipientAddress        common.Address
}

// NewProcessor creates a new blockchain processor
func NewProcessor(metadata *ServiceMetadata,orgMetaData *OrganizationMetaData) (Processor, error) {
	// TODO(aiden) accept configuration as a parameter

	p := Processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
		enabled:            config.GetBool(config.BlockchainEnabledKey),
	}

	if !p.enabled {
		return p, nil
	}

	// Setup ethereum client

	if ethclients, err := GetEthereumClient(); err != nil {
		return p, errors.Wrap(err, "error creating RPC client")
	} else {
		p.rawClient = ethclients.RawClient
		p.ethClient = ethclients.EthClient
	}
	p.groupId = orgMetaData.GetGroupId()
	p.recipientAddress = orgMetaData.GetPaymentAddress()

	// TODO: if address is not in config, try to load it using network

	//TODO: Read this from github

	p.escrowContractAddress = metadata.GetMpeAddress()

	if mpe, err := NewMultiPartyEscrow(p.escrowContractAddress, p.ethClient); err != nil {
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

func (processor *Processor) Enabled() (enabled bool) {
	return processor.enabled
}

func (processor *Processor) EscrowContractAddress() common.Address {
	return processor.escrowContractAddress
}

func (processor *Processor) MultiPartyEscrow() *MultiPartyEscrow {
	return processor.multiPartyEscrow
}

func (processor *Processor) CurrentBlock() (currentBlock *big.Int, err error) {
	// We have to do a raw call because the standard method of ethClient.HeaderByNumber(ctx, nil) errors on
	// unmarshaling the response currently. See https://github.com/ethereum/go-ethereum/issues/3230
	var currentBlockHex string
	if err = processor.rawClient.CallContext(context.Background(), &currentBlockHex, "eth_blockNumber"); err != nil {
		log.WithError(err).Error("error determining current block")
		return nil, fmt.Errorf("error determining current block: %v", err)
	}

	currentBlockBytes := common.FromHex(currentBlockHex)
	currentBlock = new(big.Int).SetBytes(currentBlockBytes)

	return
}

func (processor *Processor) HasIdentity() bool {
	return processor.address != ""
}

func (processor *Processor) Close() {
	processor.ethClient.Close()
	processor.rawClient.Close()
}
