package blockchain

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	log "github.com/sirupsen/logrus"
)

type Processor struct {
	ethClient          *ethclient.Client
	rawClient          *rpc.Client
	agent              *Agent
	privateKey         *ecdsa.PrivateKey
	address            string
	jobCompletionQueue chan *jobInfo
}

// NewProcessor creates a new blockchain processor
func NewProcessor() Processor {
	// TODO(aiden) accept configuration as a parameter

	p := Processor{
		jobCompletionQueue: make(chan *jobInfo, 1000),
	}

	if !config.GetBool(config.BlockchainEnabledKey) {
		return p
	}

	// Setup ethereum client
	if client, err := rpc.Dial(config.GetString(config.EthereumJsonRpcEndpointKey)); err != nil {
		// TODO(ai): return (processor, error) instead of panic
		log.WithError(err).Panic("error creating rpc client")
	} else {
		p.rawClient = client
		p.ethClient = ethclient.NewClient(client)
	}

	// Setup agent
	if a, err := NewAgent(common.HexToAddress(config.GetString(config.AgentContractAddressKey)), p.ethClient); err != nil {
		// TODO(ai): remove panic
		log.WithError(err).Panic("error instantiating agent")
	} else {
		p.agent = a
	}

	// Setup identity
	if privateKeyString := config.GetString(config.PrivateKeyKey); privateKeyString != "" {
		if privKey, err := crypto.HexToECDSA(privateKeyString); err != nil {
			// TODO(ai): remove panic
			log.WithError(err).Panic("error getting private key")
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

	return p
}

// StartLoops starts background processing for event and job completion routines
func (p Processor) StartLoop() {
	go p.processJobCompletions()
	go p.processEvents()
	go p.submitOldJobsForCompletion()
}

func (p Processor) processJobCompletions() {
	for jobInfo := range p.jobCompletionQueue {
		log := log.WithFields(log.Fields{"jobAddress": common.BytesToAddress(jobInfo.jobAddressBytes).Hex(),
			"jobSignature": hex.EncodeToString(jobInfo.jobSignatureBytes)})

		v, r, s, err := parseSignature(jobInfo.jobSignatureBytes)

		if err != nil {
			log.WithError(err).Error("error parsing job signature")
		}

		auth := bind.NewKeyedTransactor(p.privateKey)

		log.Debug("submitting transaction to complete job")
		if txn, err := p.agent.CompleteJob(&bind.TransactOpts{
			From:     common.HexToAddress(p.address),
			Signer:   auth.Signer,
			GasLimit: 1000000}, common.BytesToAddress(jobInfo.jobAddressBytes), v, r, s); err != nil {
			log.WithError(err).Error("error submitting transaction to complete job")
		} else {
			isPending := true

			for {
				if _, isPending, _ = p.ethClient.TransactionByHash(context.Background(), txn.Hash()); !isPending {
					break
				}
				time.Sleep(time.Second * 1)
			}
		}
	}
}

func (p Processor) processEvents() {
	sleepSecs := config.GetDuration(config.PollSleepSecsKey)
	agentContractAddress := config.GetString(config.AgentContractAddressKey)

	a, err := abi.JSON(strings.NewReader(AgentABI))

	if err != nil {
		log.WithError(err).Error("error parsing agent ABI")
		return
	}

	jobCreatedId := a.Events["JobCreated"].Id()
	jobFundedId := a.Events["JobFunded"].Id()
	jobCompletedId := a.Events["JobCompleted"].Id()

	for {
		time.Sleep(time.Second * sleepSecs)

		// We have to do a raw call because the standard method of ethClient.HeaderByNumber(ctx, nil) errors on
		// unmarshaling the response currently. See https://github.com/ethereum/go-ethereum/issues/3230
		var currentBlockHex string
		if err = p.rawClient.CallContext(context.Background(), &currentBlockHex, "eth_blockNumber"); err != nil {
			log.WithError(err).Error("error determining current block")
			continue
		}

		currentBlockBytes := common.FromHex(currentBlockHex)
		currentBlock := new(big.Int).SetBytes(currentBlockBytes)

		lastBlock := new(big.Int).Sub(currentBlock, new(big.Int).SetUint64(1))
		db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(db.ChainBucketName)
			lastBlockBytes := bucket.Get([]byte("lastBlock"))
			if lastBlockBytes != nil {
				lastBlock = new(big.Int).SetBytes(lastBlockBytes)
			}
			return nil
		})

		// Don't re-scan lastBlock
		fromBlock := new(big.Int).Add(lastBlock, new(big.Int).SetUint64(1))

		// If fromBlock <= currentBlock
		// TODO(aiden) invert logic and early return
		if fromBlock.Cmp(currentBlock) <= 0 {
			if jobCreatedLogs, err := p.ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
				FromBlock: fromBlock,
				ToBlock:   currentBlock,
				Addresses: []common.Address{common.HexToAddress(agentContractAddress)},
				Topics:    [][]common.Hash{{jobCreatedId}}}); err == nil {
				if len(jobCreatedLogs) > 0 {
					db.Update(func(tx *bolt.Tx) error {
						bucket := tx.Bucket(db.JobBucketName)
						for _, jobCreatedLog := range jobCreatedLogs {
							job := &db.Job{}
							jobAddressBytes := common.BytesToAddress(jobCreatedLog.Data[0:32]).Bytes()
							jobConsumerBytes := common.BytesToAddress(jobCreatedLog.Data[32:64]).Bytes()

							log.WithFields(log.Fields{
								"jobAddress": common.BytesToAddress(jobAddressBytes).Hex(),
							}).Debug("received JobCreated event; saving to db")

							jobBytes := bucket.Get(jobAddressBytes)
							if jobBytes != nil {
								json.Unmarshal(jobBytes, job)
							}
							job.JobAddress = jobAddressBytes
							job.Consumer = jobConsumerBytes
							job.JobState = JobPendingState
							if jobBytes, err := json.Marshal(job); err == nil {
								if err = bucket.Put(jobAddressBytes, jobBytes); err != nil {
									log.WithError(err).Error("error putting job to db")
								}
							} else {
								log.WithError(err).Error("error marshaling job")
							}
						}
						return nil
					})
				}
			} else {
				log.WithError(err).Error("error getting job created logs")
			}

			if jobFundedLogs, err := p.ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
				FromBlock: fromBlock,
				ToBlock:   currentBlock,
				Addresses: []common.Address{common.HexToAddress(agentContractAddress)},
				Topics:    [][]common.Hash{{jobFundedId}}}); err == nil {
				if len(jobFundedLogs) > 0 {
					db.Update(func(tx *bolt.Tx) error {
						bucket := tx.Bucket(db.JobBucketName)
						for _, jobFundedLog := range jobFundedLogs {
							job := &db.Job{}
							jobAddressBytes := common.BytesToAddress(jobFundedLog.Data[0:32]).Bytes()

							log.WithFields(log.Fields{
								"jobAddress": common.BytesToAddress(jobAddressBytes).Hex(),
							}).Debug("received JobFunded event; saving to db")

							jobBytes := bucket.Get(jobAddressBytes)
							if jobBytes != nil {
								json.Unmarshal(jobBytes, job)
							}
							job.JobAddress = jobAddressBytes
							job.JobState = JobFundedState
							if jobBytes, err := json.Marshal(job); err == nil {
								if err = bucket.Put(jobAddressBytes, jobBytes); err != nil {
									log.WithError(err).Error("error putting job to db")
								}
							} else {
								log.WithError(err).Error("error marshaling job")
							}
						}
						return nil
					})
				}
			} else {
				log.WithError(err).Error("error getting job funded logs")
			}

			if jobCompletedLogs, err := p.ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
				FromBlock: fromBlock,
				ToBlock:   currentBlock,
				Addresses: []common.Address{common.HexToAddress(agentContractAddress)},
				Topics:    [][]common.Hash{{jobCompletedId}}}); err == nil {
				if len(jobCompletedLogs) > 0 {
					db.Update(func(tx *bolt.Tx) error {
						bucket := tx.Bucket(db.JobBucketName)
						for _, jobCompletedLog := range jobCompletedLogs {
							jobAddressBytes := common.BytesToAddress(jobCompletedLog.Data[0:32]).Bytes()

							log.WithFields(log.Fields{
								"jobAddress": common.BytesToAddress(jobAddressBytes).Hex(),
							}).Debug("received JobCompleted event; deleting from db")

							if err = bucket.Delete(jobAddressBytes); err != nil {
								log.WithError(err).Error("error deleting job from db")
							}
						}
						return nil
					})
				}
			} else {
				log.WithError(err).Error("error getting job completed logs")
			}

			db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(db.ChainBucketName)
				if err = bucket.Put([]byte("lastBlock"), currentBlockBytes); err != nil {
					log.WithError(err).Error("error putting current block to db")
				}
				return nil
			})
		}
	}
}

func (p Processor) submitOldJobsForCompletion() {
	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(db.JobBucketName)
		bucket.ForEach(func(k, v []byte) error {
			job := &db.Job{}
			json.Unmarshal(v, job)
			if job.Completed {
				log.WithFields(log.Fields{
					"jobAddress":   common.BytesToAddress(job.JobAddress).Hex(),
					"jobSignature": hex.EncodeToString(job.JobSignature),
				}).Debug("completing old job found in db")
				p.jobCompletionQueue <- &jobInfo{job.JobAddress, job.JobSignature}
			}
			return nil
		})
		return nil
	})
}
