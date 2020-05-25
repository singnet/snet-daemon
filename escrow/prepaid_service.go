package escrow

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/big"
	"time"
)

type lockingPrepaidService struct {
	storage        *PrepaidStorage
	validator      *PrePaidPaymentValidator
	replicaGroupID func() ([32]byte, error)
}

func NewPrePaidService(
	storage *PrepaidStorage,
	prepaidValidator *PrePaidPaymentValidator, groupIdReader func() ([32]byte, error)) PrePaidService {
	return &lockingPrepaidService{
		storage:        storage,
		validator:      prepaidValidator,
		replicaGroupID: groupIdReader,
	}
}

func (h *lockingPrepaidService) ListPrePaidUsers() (users []*PrePaidUsageData, err error) {
	return h.storage.GetAll()
}

func (h *lockingPrepaidService) GetPrePaidUserKey(payment *PrePaidPayment) (userKey *PrePaidUserKey, err error) {
	return nil, nil
}

func (h *lockingPrepaidService) GetPrePaidUser(key *PrePaidUserKey) (PrePaidUser *PrePaidUsageData, ok bool, err error) {
	return h.storage.Get(key.ID())
}

//todo discuss if there is a better approach with Vitaly !
//https://github.com/datawisesystems/etcd-lock/blob/master/rwlock.go
func (h *lockingPrepaidService) UpdateUsage(key *PrePaidUserKey, usage UpdateUsage, revisedAmount *big.Int) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	lockKey := fmt.Sprintf("lock-channel-%v", key.ID())
	defer cancel()
	mu, err := h.Mutex(lockKey)
	if err != nil {
		log.WithError(err).Error("Unable to Get lock on mutex")
		return
	}

	//Now we have a Lock
	//Get the Storage
	oldValue, ok, err := h.storage.Get(key.ID())
	if err != nil {
		log.WithError(err).Error("unable to get usage from pre paid storage for key", key)
		return
	}

	if !ok {
		return fmt.Errorf("Channel ID %v is not set for pre paid usage")
	}

	newValue, err := usage(oldValue, revisedAmount)
	//Check with Vitaly , if there is a better way of doing this !
	ok, err = h.storage.CompareAndSwap(key, oldValue, newValue)
	if err != nil {
		log.WithError(err).Error("unable to CompareAndSwap pre paid storage for key", key)
		return
	}

	if !ok {
		return fmt.Errorf("unable to CompareAndSwap pre paid storage for k", key)
	}

	defer func() {
		if r := recover(); r != nil {
			log.WithField("recover", r).Warn("PrePaid Service , Error on UpdateUsage")
		}
		err = mu.Unlock(ctx)
		if err != nil {
			log.WithError(err).Error("Error during mutex Unlock at PrePaid Service , Error on UpdateUsage")
			return
		}

	}()
	return nil
}

func (h *lockingPrepaidService) Mutex(key string) (mutex CustomMutex, err error) {
	return h.storage.GetMutex(key)
}
