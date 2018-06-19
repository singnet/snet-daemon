package db

import (
	"github.com/coreos/bbolt"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
)

type Job struct {
	JobAddress   []byte
	JobSignature []byte
	JobState     string
	Consumer     []byte
	Completed    bool
}

var (
	JobBucketName   = []byte("job")
	ChainBucketName = []byte("chain")
	database        *bolt.DB
)

func init() {
	if config.GetBool(config.BlockchainEnabledKey) {
		db, err := bolt.Open(config.GetString(config.DbPathKey), 0644, nil)
		if err != nil {
			log.Panicf("error opening db; error: %+v", err)
		}
		database = db
		if err = database.Update(func(tx *bolt.Tx) error {
			if _, err = tx.CreateBucketIfNotExists(JobBucketName); err != nil {
				return err
			}
			_, err = tx.CreateBucketIfNotExists(ChainBucketName)
			return err
		}); err != nil {
			log.Panicf("error initializing db; error: %+v")
		}
	}
}

func Shutdown() {
	if config.GetBool(config.BlockchainEnabledKey) {
		database.Close()
	}
}

func Update(fn func(tx *bolt.Tx) error) error {
	return database.Update(fn)
}

func View(fn func(tx *bolt.Tx) error) error {
	return database.View(fn)
}
