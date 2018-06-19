//go:generate abigen --abi ../resources/Agent.abi --pkg blockchain --type Agent --out agent.go

package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
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

type jobInfo struct {
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

var (
	ethClient          *ethclient.Client
	rawClient          *rpc.Client
	agent              *Agent
	privateKey         *ecdsa.PrivateKey
	address            string
	jobCompletionQueue chan *jobInfo
)

func init() {
	if config.GetBool(config.BlockchainEnabledKey) {
		// Setup ethereum client
		if client, err := rpc.Dial(config.GetString(config.EthereumJsonRpcEndpointKey)); err != nil {
			log.WithError(err).Panic("error creating rpc client")
		} else {
			rawClient = client
			ethClient = ethclient.NewClient(rawClient)
		}

		// Setup agent
		if a, err := NewAgent(common.HexToAddress(config.GetString(config.AgentContractAddressKey)), ethClient); err != nil {
			log.WithError(err).Panic("error instantiating agent")
		} else {
			agent = a
		}

		// Setup identity
		if privateKeyString := config.GetString(config.PrivateKeyKey); privateKeyString != "" {
			if privKey, err := crypto.HexToECDSA(privateKeyString); err != nil {
				log.WithError(err).Panic("error getting private key")
			} else {
				privateKey = privKey
				address = crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
			}
		} else if hdwalletMnemonic := config.GetString(config.HdwalletMnemonicKey); hdwalletMnemonic != "" {
			if privKey, err := derivePrivateKey(hdwalletMnemonic, 44, 60, 0, 0, uint32(config.GetInt(config.HdwalletIndexKey))); err != nil {
				log.WithError(err).Panic("error deriving private key")
			} else {
				privateKey = privKey
				address = crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
			}
		}

		// Start event and job completion routines
		jobCompletionQueue = make(chan *jobInfo, 1000)
		go processJobCompletions()
		go processEvents()
		go submitOldJobsForCompletion()
	}
}

func GetGrpcStreamInterceptor() grpc.StreamServerInterceptor {
	if config.GetBool(config.BlockchainEnabledKey) {
		return jobValidationInterceptor
	} else {
		return noOpInterceptor
	}
}

func IsValidJobInvocation(jobAddressBytes, jobSignatureBytes []byte) bool {
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

	pubKey, err := crypto.SigToPub(crypto.Keccak256([]byte{0x19}, []byte("Ethereum Signed Message:\n32"),
		crypto.Keccak256(jobAddressBytes)), bytes.Join([][]byte{jobSignatureBytes[0:64], {v % 27}}, []byte{}))
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
	if validated, err := agent.ValidateJobInvocation(&bind.CallOpts{
		Pending: true,
		From:    common.HexToAddress(address)}, common.BytesToAddress(jobAddressBytes), v, r, s); err != nil {
		log.WithError(err).Error("error validating job on chain")
		return false
	} else if !validated {
		log.Debug("job failed to validate")
		return false
	}

	log.Debug("validated job invocation on chain")
	return true
}

func CompleteJob(jobAddressBytes, jobSignatureBytes []byte) {
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
	jobCompletionQueue <- &jobInfo{jobAddressBytes, jobSignatureBytes}
}
