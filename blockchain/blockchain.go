//go:generate abigen --abi ../resources/blockchain/node_modules/singularitynet-platform-contracts/abi/Agent.json --pkg blockchain --type Agent --out agent.go
//go:generate abigen --abi ../resources/blockchain/node_modules/singularitynet-platform-contracts/abi/MultiPartyEscrow.json --bin ../resources/blockchain/node_modules/singularitynet-platform-contracts/bytecode/MultiPartyEscrow.hex --pkg blockchain --type MultiPartyEscrow --out multi_party_escrow.go

package blockchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/md5"
	"encoding/hex"

	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
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
	enabled               bool
	ethClient             *ethclient.Client
	rawClient             *rpc.Client
	agent                 *Agent
	sigHasher             func([]byte) []byte
	privateKey            *ecdsa.PrivateKey
	address               string
	jobCompletionQueue    chan *jobInfo
	boltDB                *bolt.DB
	escrowContractAddress common.Address
}

// NewProcessor creates a new blockchain processor
func NewProcessor(boltDB *bolt.DB) (Processor, error) {
	// TODO(aiden) accept configuration as a parameter

	p := Processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
		enabled:            config.GetBool(config.BlockchainEnabledKey),
		boltDB:             boltDB,
	}

	if !p.enabled {
		return p, nil
	}

	// Setup ethereum client
	if client, err := rpc.Dial(config.GetString(config.EthereumJsonRpcEndpointKey)); err != nil {
		return p, errors.Wrap(err, "error creating RPC client")
	} else {
		p.rawClient = client
		p.ethClient = ethclient.NewClient(client)
	}

	// TODO: if address is not in config, try to load it using network
	// configuration
	p.escrowContractAddress = common.HexToAddress(config.GetString(config.MultiPartyEscrowContractAddressKey))
	agentAddress := common.HexToAddress(config.GetString(config.AgentContractAddressKey))

	// Setup agent
	if a, err := NewAgent(agentAddress, p.ethClient); err != nil {
		return p, errors.Wrap(err, "error instantiating agent")
	} else {
		p.agent = a
	}

	// Determine "version" of agent contract and set local signature hash creator
	if bytecode, err := p.ethClient.CodeAt(context.Background(), agentAddress, nil); err != nil {
		return p, errors.Wrap(err, "error retrieving agent bytecode")
	} else {
		bcSum := md5.Sum(bytecode)

		// Compare checksum of agent with known checksum of the first version of the agent contract, which signed
		// the raw bytes of the address rather than the hex-encoded string
		if bytes.Equal(bcSum[:], []byte{244, 176, 168, 6, 74, 56, 171, 175, 38, 48, 245, 246, 189, 0, 67, 200}) {
			p.sigHasher = func(i []byte) []byte {
				return crypto.Keccak256(HashPrefix32Bytes, crypto.Keccak256(i))
			}
		} else {
			p.sigHasher = func(i []byte) []byte {
				return crypto.Keccak256(hashPrefix42Bytes, []byte(hex.EncodeToString(i)))
			}
		}
	}

	// Setup identity
	if privateKeyString := config.GetString(config.PrivateKeyKey); privateKeyString != "" {
		if privKey, err := crypto.HexToECDSA(privateKeyString); err != nil {
			return p, errors.Wrap(err, "error getting private key")
		} else {
			p.privateKey = privKey
			p.address = crypto.PubkeyToAddress(p.privateKey.PublicKey).Hex()
		}
	} else if hdwalletMnemonic := config.GetString(config.HdwalletMnemonicKey); hdwalletMnemonic != "" {
		if privKey, err := derivePrivateKey(hdwalletMnemonic, 44, 60, 0, 0, uint32(config.GetInt(config.HdwalletIndexKey))); err != nil {
			log.WithError(err).Panic("error deriving private key")
		} else {
			p.privateKey = privKey
			p.address = crypto.PubkeyToAddress(p.privateKey.PublicKey).Hex()
		}
	}

	return p, nil
}

func (processor *Processor) Enabled() (enabled bool) {
	return processor.enabled
}

func (processor *Processor) EscrowContractAddress() common.Address {
	return processor.escrowContractAddress
}
