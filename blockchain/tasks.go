package blockchain

import (
	"context"
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
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/db"
	log "github.com/sirupsen/logrus"
)

func processJobCompletions() {
	for jobInfo := range jobCompletionQueue {
		log := log.WithFields(log.Fields{"jobAddress": common.BytesToAddress(jobInfo.jobAddressBytes).Hex(),
			"jobSignature": hex.EncodeToString(jobInfo.jobSignatureBytes)})

		v, r, s, err := parseSignature(jobInfo.jobSignatureBytes)

		if err != nil {
			log.WithError(err).Error("error parsing job signature")
		}

		auth := bind.NewKeyedTransactor(privateKey)

		log.Debug("submitting transaction to complete job")
		if txn, err := agent.CompleteJob(&bind.TransactOpts{
			From:     common.HexToAddress(address),
			Signer:   auth.Signer,
			GasLimit: 1000000}, common.BytesToAddress(jobInfo.jobAddressBytes), v, r, s); err != nil {
			log.WithError(err).Error("error submitting transaction to complete job")
		} else {
			isPending := true

			for {
				if _, isPending, _ = ethClient.TransactionByHash(context.Background(), txn.Hash()); !isPending {
					break
				}
				time.Sleep(time.Second * 1)
			}
		}
	}
}

func processEvents() {
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
		if err = rawClient.CallContext(context.Background(), &currentBlockHex, "eth_blockNumber"); err != nil {
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
		if fromBlock.Cmp(currentBlock) <= 0 {
			if jobCreatedLogs, err := ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
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

			if jobFundedLogs, err := ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
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

			if jobCompletedLogs, err := ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
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

func submitOldJobsForCompletion() {
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
				jobCompletionQueue <- &jobInfo{job.JobAddress, job.JobSignature}
			}
			return nil
		})
		return nil
	})
}
