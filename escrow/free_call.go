package escrow

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
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

func (transaction *freeCallTransaction) GetSender() common.Address {
	return common.HexToAddress(transaction.payment.Address)
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
		ServiceId: payment.ServiceId, GroupID: utils.BytesToBase64(groupId[:])}, err
}

// StartFreeCallUserTransaction acquires a user-level lock and returns a transaction
// handle for free-call. It validates the per-address/global free-call
// limits (where -1 means unlimited) and fails if the limit is exceeded.
// The returned transaction keeps the lock and must be finalized elsewhere.
func (h *lockingFreeCallUserService) StartFreeCallUserTransaction(payment *FreeCallPayment) (transaction FreeCallTransaction, err error) {

	userKey, err := h.GetFreeCallUserKey(payment)
	if err != nil {
		return nil, NewPaymentError(Internal, "payment freeCallUserKey error: %s", err.Error())
	}

	freeCallUserData, ok, err := h.FreeCallUser(userKey)
	if err != nil {
		return nil, NewPaymentError(Internal, "payment freeCallUserData error: %s", err.Error())
	}

	if !ok {
		zap.L().Warn("Payment freeCallUserData not found")
		return nil, NewPaymentError(Unauthenticated, "payment freeCallUserData \"%v\" not found", userKey)
	}

	freeCallUserData.ServiceId = userKey.ServiceId
	freeCallUserData.OrganizationId = userKey.OrganizationId
	freeCallUserData.GroupID = userKey.GroupID
	freeCallUserData.Address = userKey.Address

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
				// todo send a notification to the developer ( contact email is in service metadata)
				zap.L().Error("Transaction is cancelled because of err, but freeCallUserData cannot be unlocked. All other transactions on this freeCallUserData will be blocked until unlock. Please unlock freeCallUserData manually.",
					zap.Any("userKey", userKey),
					zap.Error(err))
			}
		}
	}(lock)

	// Check if free calls are allowed for this user
	allowed := config.GetFreeCallsAllowed(userKey.Address)
	if allowed == 0 {
		allowed = h.serviceMetadata.GetFreeCallsAllowed() // meta is >= 0 by contract
	}

	if allowed != -1 {
		made := freeCallUserData.FreeCallsMade
		if made >= allowed {
			return nil, fmt.Errorf(
				"free call limit has been exceeded, calls made = %d, total free calls eligible = %d",
				made, allowed,
			)
		}
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
