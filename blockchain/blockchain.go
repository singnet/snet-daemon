//go:generate abigen --abi ../resources/Agent.abi --pkg blockchain --type Agent --out agent.go

package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

func (p Processor) GrpcStreamInterceptor() grpc.StreamServerInterceptor {
	if config.GetBool(config.BlockchainEnabledKey) {
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
