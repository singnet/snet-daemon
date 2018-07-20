//go:generate abigen --abi ../resources/blockchain/node_modules/singularitynet-platform-contracts/abi/Agent.json --pkg blockchain --type Agent --out agent.go

package blockchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	JobPendingState    = "PENDING"
	JobFundedState     = "FUNDED"
	JobAddressHeader   = "snet-job-address"
	JobSignatureHeader = "snet-job-signature"
)

var (
	// Ethereum signature prefix: see https://github.com/ethereum/go-ethereum/blob/bf468a81ec261745b25206b2a596eb0ee0a24a74/internal/ethapi/api.go#L361
	hashPrefix32Bytes = []byte("\x19Ethereum Signed Message:\n32")
	hashPrefix42Bytes = []byte("\x19Ethereum Signed Message:\n420x")
)

type jobInfo struct {
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

type Processor struct {
	enabled            bool
	ethClient          *ethclient.Client
	rawClient          *rpc.Client
	agent              *Agent
	sigHasher          func([]byte) []byte
	privateKey         *ecdsa.PrivateKey
	address            string
	jobCompletionQueue chan *jobInfo
}

// NewProcessor creates a new blockchain processor
func NewProcessor() (Processor, error) {
	// TODO(aiden) accept configuration as a parameter

	p := Processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
		enabled:            config.GetBool(config.BlockchainEnabledKey),
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
				return crypto.Keccak256(hashPrefix32Bytes, crypto.Keccak256(i))
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

func (p Processor) GrpcStreamInterceptor() grpc.StreamServerInterceptor {
	if p.enabled {
		return p.jobValidationInterceptor
	} else {
		return noOpInterceptor
	}
}

func (p Processor) IsValidJobInvocation(jobAddressBytes, jobSignatureBytes []byte) bool {
	log := log.WithFields(log.Fields{
		"jobAddress":   common.BytesToAddress(jobAddressBytes).Hex(),
		"jobSignature": hex.EncodeToString(jobSignatureBytes)})

	// Get job state from db
	log.Debug("retrieving job from database")
	job := &db.Job{}

	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(db.JobBucketName)
		jobBytes := bucket.Get(jobAddressBytes)
		if jobBytes != nil {
			json.Unmarshal(jobBytes, job)
		}
		return nil
	})

	// If job is marked completed locally, reject
	if job.Completed {
		log.Debug("job already marked completed locally")
		return false
	}

	v, r, s, err := parseSignature(jobSignatureBytes)
	if err != nil {
		log.WithError(err).Error("error parsing signature")
		return false
	}

	pubKey, err := crypto.SigToPub(p.sigHasher(jobAddressBytes), bytes.Join([][]byte{jobSignatureBytes[0:64], {v % 27}},
		[]byte{}))
	if err != nil {
		log.WithError(err).Error("error recovering signature")
		return false
	}

	// If job is FUNDED and signature validates, skip on-chain validation
	if job.JobState == JobFundedState && bytes.Equal(crypto.PubkeyToAddress(*pubKey).Bytes(), job.Consumer) {
		log.Debug("validated job invocation locally")
		return true
	}

	log.Debug("unable to validate job invocation locally; falling back to on-chain validation")

	// Fall back to on-chain validation
	if validated, err := p.agent.ValidateJobInvocation(&bind.CallOpts{
		Pending: true,
		From:    common.HexToAddress(p.address)}, common.BytesToAddress(jobAddressBytes), v, r, s); err != nil {
		log.WithError(err).Error("error validating job on chain")
		return false
	} else if !validated {
		log.Debug("job failed to validate")
		return false
	}

	log.Debug("validated job invocation on chain")
	return true
}

func (p Processor) CompleteJob(jobAddressBytes, jobSignatureBytes []byte) {
	job := &db.Job{}

	// Mark the job completed in the db synchronously
	if err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(db.JobBucketName)
		jobBytes := bucket.Get(jobAddressBytes)
		if jobBytes != nil {
			json.Unmarshal(jobBytes, job)
		}
		job.Completed = true
		job.JobSignature = jobSignatureBytes
		jobBytes, err := json.Marshal(job)
		if err != nil {
			return err
		}
		if err = bucket.Put(jobAddressBytes, jobBytes); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.WithFields(log.Fields{
			"jobAddress":   common.BytesToAddress(jobAddressBytes).Hex(),
			"jobSignature": hex.EncodeToString(jobSignatureBytes),
		}).WithError(err).Error("error marking job completed in db")
	}

	// Submit the job for completion
	p.jobCompletionQueue <- &jobInfo{jobAddressBytes, jobSignatureBytes}
}
