package escrow

import (
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	log "github.com/sirupsen/logrus"
)

type lockingFreeCallUserService struct {
	storage         *FreeCallUserStorage
	locker          Locker
	replicaGroupID  func() ([32]byte, error)
	serviceMetadata *blockchain.ServiceMetadata
}

func NewFreeCallUserService(
	storage *FreeCallUserStorage,

	locker Locker,
	groupIdReader func() ([32]byte, error), metadata *blockchain.ServiceMetadata) FreeCallUserService {

	return &lockingFreeCallUserService{
		storage:         storage,
		locker:          locker,
		replicaGroupID:  groupIdReader,
		serviceMetadata: metadata,
	}
}

func (h *lockingFreeCallUserService) FreeCallUser(key *FreeCallUserKey) (freeCallUser *FreeCallUserData, ok bool, err error) {
	freeCallUser, ok, err = h.storage.Get(key)
	if err != nil {
		return
	}
	if !ok {
		return &FreeCallUserData{FreeCallsMade: 0,UserId:key.UserId}, true, nil
	}
	//the below was added only to set the userid for historic entries, will be removed soon todo
	freeCallUser.UserId = key.UserId
	return
}

func (h *lockingFreeCallUserService) ListFreeCallUsers() (users []*FreeCallUserData, err error) {
	return h.storage.GetAll()
}

type freeCallTransaction struct {
	payment         FreeCallPayment
	freeCallUser    *FreeCallUserData
	freeCallUserKey *FreeCallUserKey
	service         *lockingFreeCallUserService
	lock            Lock
}

func (transaction *freeCallTransaction) String() string {
	return fmt.Sprintf("{FreeCallPayment: %v, FreeCallUser: %v}", transaction.payment.String(), transaction.freeCallUser.String())
}

func (transaction *freeCallTransaction) FreeCallUser() *FreeCallUserData {
	return transaction.freeCallUser
}

func (h *lockingFreeCallUserService) GetFreeCallUserKey(payment *FreeCallPayment) (userKey *FreeCallUserKey, err error) {
	groupId, err := h.replicaGroupID()
	return &FreeCallUserKey{UserId: payment.UserId, OrganizationId: payment.OrganizationId,
		ServiceId: payment.ServiceId, GroupID: blockchain.BytesToBase64(groupId[:])}, err
}
func (h *lockingFreeCallUserService) StartFreeCallUserTransaction(payment *FreeCallPayment) (transaction FreeCallTransaction, err error) {
	userKey, err := h.GetFreeCallUserKey(payment)
	if err != nil {
		return nil, NewPaymentError(Internal, "payment freeCallUserKey error:"+err.Error())
	}
	freeCallUserData, ok, err := h.FreeCallUser(userKey)

	if err != nil {
		return nil, NewPaymentError(Internal, "payment freeCallUserData error:"+err.Error())
	}
	if !ok {
		log.Warn("Payment freeCallUserData not found")
		return nil, NewPaymentError(Unauthenticated, "payment freeCallUserData \"%v\" not found", userKey)
	}

	lock, ok, err := h.locker.Lock(userKey.String())
	if err != nil {
		return nil, NewPaymentError(Internal, "cannot get mutex for user: %v", userKey)
	}
	if !ok {
		return nil, NewPaymentError(FailedPrecondition, "another transaction on this user: %v is in progress", userKey)
	}
	defer func(lock Lock) {
		if err != nil {
			e := lock.Unlock()
			if e != nil {
				//todo send a notification to the devloper ( contact email is in service metadata)
				log.WithError(e).WithField("userKey", userKey).WithField("err", err).Error("Transaction is cancelled because of err, but freeCallUserData cannot be unlocked. All other transactions on this freeCallUserData will be blocked until unlock. Please unlock freeCallUserData manually.")
			}
		}
	}(lock)

	//Check if free calls are allowed or not on this user
	if freeCallUserData.FreeCallsMade >= h.serviceMetadata.GetFreeCallsAllowed() {
		return nil, fmt.Errorf("free call limit has been exceeded, calls made "+
			"= %v,total free calls eligible = %v", freeCallUserData.FreeCallsMade, h.serviceMetadata.GetFreeCallsAllowed())
	}
	return &freeCallTransaction{
		payment:         *payment,
		freeCallUserKey: userKey,
		freeCallUser:    freeCallUserData,
		lock:            lock,
		service:         h,
	}, nil
}

func (transaction *freeCallTransaction) Commit() error {
	defer func(payment *freeCallTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			//todo send a notification to the developer defined in service metadata
			log.WithError(err).WithField("transaction", payment).
				Error("free call user cannot be unlocked because of error." +
					" All other transactions on this channel will be blocked until unlock." +
					" Please unlock user for free calls manually.")
		} else {
			log.Debug("free call user unlocked")
		}
	}(transaction)

	IncrementFreeCallCount(transaction.FreeCallUser())
	e := transaction.service.storage.Put(
		transaction.freeCallUserKey,
		transaction.FreeCallUser(),
	)
	if e != nil {
		log.WithError(e).Error("Unable to store new transaction free call user state")
		return NewPaymentError(Internal, "unable to store new transaction free call user state")
	}

	log.Debug("Free Call Payment completed")
	return nil
}

func (payment *freeCallTransaction) Rollback() error {
	defer func(payment *freeCallTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			log.WithError(err).WithField("payment", payment).Error("free call user cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.")
		} else {
			log.Debug("Free call Payment rolled back, free call user unlocked")
		}
	}(payment)
	return nil
}
