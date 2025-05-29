package escrow

import (
	"fmt"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"go.uber.org/zap"
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
		return &FreeCallUserData{FreeCallsMade: 0, UserID: key.UserId, Address: key.Address, GroupID: key.GroupID, ServiceId: key.ServiceId, OrganizationId: key.OrganizationId}, true, nil
	}
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
	return &FreeCallUserKey{UserId: payment.UserID, Address: payment.Address, OrganizationId: payment.OrganizationId,
		ServiceId: payment.ServiceId, GroupID: blockchain.BytesToBase64(groupId[:])}, err
}

func (h *lockingFreeCallUserService) StartFreeCallUserTransaction(payment *FreeCallPayment) (transaction FreeCallTransaction, err error) {
	userKey, err := h.GetFreeCallUserKey(payment)
	if err != nil {
		return nil, NewPaymentError(Internal, "payment freeCallUserKey error: %s", err.Error())
	}
	freeCallUserData, ok, err := h.FreeCallUser(userKey)
	//todo , will remove this line once all data is re initialized
	freeCallUserData.ServiceId = userKey.ServiceId
	freeCallUserData.OrganizationId = userKey.OrganizationId
	freeCallUserData.GroupID = userKey.GroupID

	if err != nil {
		return nil, NewPaymentError(Internal, "payment freeCallUserData error: %s", err.Error())
	}
	if !ok {
		zap.L().Warn("Payment freeCallUserData not found")
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
				//todo send a notification to the developer ( contact email is in service metadata)
				zap.L().Error("Transaction is cancelled because of err, but freeCallUserData cannot be unlocked. All other transactions on this freeCallUserData will be blocked until unlock. Please unlock freeCallUserData manually.",
					zap.Any("userKey", userKey),
					zap.Error(err))
			}
		}
	}(lock)

	var countFreeCallsAllowed int
	if countFreeCallsAllowed = config.GetFreeCallsAllowed(freeCallUserData.Address); countFreeCallsAllowed <= 0 {
		countFreeCallsAllowed = h.serviceMetadata.GetFreeCallsAllowed()
	}

	// Check if free calls are allowed or not on this user
	if freeCallUserData.FreeCallsMade >= countFreeCallsAllowed {
		return nil, fmt.Errorf("free call limit has been exceeded, calls made "+
			"= %v,total free calls eligible = %v", freeCallUserData.FreeCallsMade, countFreeCallsAllowed)
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
			zap.L().Error("free call user cannot be unlocked because of error."+
				" All other transactions on this channel will be blocked until unlock."+
				" Please unlock user for free calls manually.", zap.Any("transaction", payment), zap.Error(err))
		} else {
			zap.L().Debug("free call user unlocked")
		}
	}(transaction)

	IncrementFreeCallCount(transaction.FreeCallUser())
	err := transaction.service.storage.Put(
		transaction.freeCallUserKey,
		transaction.FreeCallUser(),
	)
	if err != nil {
		zap.L().Error("Unable to store new transaction free call user state")
		return NewPaymentError(Internal, "unable to store new transaction free call user state")
	}

	zap.L().Debug("Free Call Payment completed")
	return nil
}

func (transaction *freeCallTransaction) Rollback() error {
	defer func(payment *freeCallTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			zap.L().Error("free call user cannot be unlocked because of error. All other transactions on this channel will be blocked until unlock. Please unlock channel manually.",
				zap.Error(err), zap.Any("payment", payment))
		} else {
			zap.L().Debug("Free call Payment rolled back, free call user unlocked")
		}
	}(transaction)
	return nil
}
