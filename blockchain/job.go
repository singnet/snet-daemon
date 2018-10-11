package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/db"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	jobPendingState = "PENDING"
	jobFundedState  = "FUNDED"

	JobAddressHeader   = "snet-job-address"
	JobSignatureHeader = "snet-job-signature"

	// JobPaymentType each call should be payed using unique instance of funded Job
	JobPaymentType = "job"
)

type jobPaymentHandler struct {
	p *Processor
}

type jobPaymentType struct {
	grpcContext       *GrpcStreamContext
	jobAddressBytes   []byte
	jobSignatureBytes []byte
}

// NewJobPaymentHandler returns an instance of PaymentHandler which validates
// payments through Job contract.
func NewJobPaymentHandler(p *Processor) PaymentHandler {
	return &jobPaymentHandler{p: p}
}

func (h *jobPaymentHandler) Type() (typ string) {
	return JobPaymentType
}

func (h *jobPaymentHandler) Payment(context *GrpcStreamContext) (payment Payment, err *status.Status) {
	jobAddressBytes, err := GetBytesFromHex(context.MD, JobAddressHeader)
	if err != nil {
		return
	}

	jobSignatureBytes, err := GetBytesFromHex(context.MD, JobSignatureHeader)
	if err != nil {
		return
	}

	return &jobPaymentType{
		grpcContext:       context,
		jobAddressBytes:   jobAddressBytes,
		jobSignatureBytes: jobSignatureBytes,
	}, nil
}

func (h *jobPaymentHandler) Validate(_payment Payment) (err *status.Status) {
	var payment = _payment.(*jobPaymentType)

	valid := h.p.IsValidJobInvocation(payment.jobAddressBytes, payment.jobSignatureBytes)
	if !valid {
		return status.Newf(codes.Unauthenticated, "job invocation not valid")
	}

	return nil
}

func (h *jobPaymentHandler) Complete(_payment Payment) (err *status.Status) {
	var payment = _payment.(*jobPaymentType)
	h.p.CompleteJob(payment.jobAddressBytes, payment.jobSignatureBytes)
	return nil
}

func (h *jobPaymentHandler) CompleteAfterError(_payment Payment, result error) (err *status.Status) {
	return nil
}

func (p *Processor) IsValidJobInvocation(jobAddressBytes, jobSignatureBytes []byte) bool {
	log := log.WithFields(log.Fields{
		"jobAddress":   common.BytesToAddress(jobAddressBytes).Hex(),
		"jobSignature": hex.EncodeToString(jobSignatureBytes)})

	// Get job state from db
	log.Debug("retrieving job from database")
	job := &db.Job{}

	p.boltDB.View(func(tx *bolt.Tx) error {
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
		nil))
	if err != nil {
		log.WithError(err).Error("error recovering signature")
		return false
	}

	// If job is FUNDED and signature validates, skip on-chain validation
	if job.JobState == jobFundedState && bytes.Equal(crypto.PubkeyToAddress(*pubKey).Bytes(), job.Consumer) {
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

func (p *Processor) CompleteJob(jobAddressBytes, jobSignatureBytes []byte) {
	job := &db.Job{}

	// Mark the job completed in the db synchronously
	if err := p.boltDB.Update(func(tx *bolt.Tx) error {
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
