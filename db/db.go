package db

import (
	"github.com/coreos/bbolt"
	"github.com/pkg/errors"
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
)

// Connect initializes a connection to the given BoltDB
func Connect(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0644, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error opening database")
	}

	if err = db.Update(func(tx *bolt.Tx) error {
		if _, err = tx.CreateBucketIfNotExists(JobBucketName); err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(ChainBucketName)
		return err
	}); err != nil {
		return nil, errors.Wrap(err, "error initializing db")
	}

	return db, nil
}
