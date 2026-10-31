package freecall

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"go.uber.org/zap"
)

type lockingUserService struct {
	storage         *UserStorage
	locker          storage.Locker
	replicaGroupID  func() ([32]byte, error)
	serviceMetadata *blockchain.ServiceMetadata
}

// NewUserService manages usage and locks in the supplied storage backend. Quotas
// come from daemon configuration or service metadata, not an MPE/Registry rule.
func NewUserService(userStorage *UserStorage, locker storage.Locker,
	groupIDReader func() ([32]byte, error), metadata *blockchain.ServiceMetadata) UserService {
	return &lockingUserService{
		storage:         userStorage,
		locker:          locker,
		replicaGroupID:  groupIDReader,
		serviceMetadata: metadata,
	}
}

func (h *lockingUserService) User(key *UserKey) (freeCallUser *UserData, ok bool, err error) {
	freeCallUser, ok, err = h.storage.Get(key)
	if err != nil {
		return
	}
	if !ok {
		return &UserData{FreeCallsMade: 0, UserID: key.UserId, Address: key.Address, GroupID: key.GroupID, ServiceId: key.ServiceId, OrganizationId: key.OrganizationId}, true, nil
	}
	return
}

func (h *lockingUserService) ListUsers() (users []*UserData, err error) {
	return h.storage.GetAll()
}

type userTransaction struct {
	payment         Payment
	freeCallUser    *UserData
	freeCallUserKey *UserKey
	service         *lockingUserService
	lock            storage.Lock
}

func (transaction *userTransaction) GetSender() common.Address {
	return common.HexToAddress(transaction.payment.Address)
}

func (transaction *userTransaction) String() string {
	return fmt.Sprintf("{Payment: %v, User: %v}", transaction.payment.String(), transaction.freeCallUser.String())
}

func (transaction *userTransaction) User() *UserData {
	return transaction.freeCallUser
}

func (h *lockingUserService) UserKey(payment *Payment) (userKey *UserKey, err error) {
	groupId, err := h.replicaGroupID()
	return &UserKey{UserId: payment.UserID, Address: payment.Address, OrganizationId: payment.OrganizationId,
		ServiceId: payment.ServiceId, GroupID: utils.BytesToBase64(groupId[:])}, err
}

// StartTransaction reads the user's usage snapshot, acquires its storage-key lock,
// and checks the allowance for that identity. A per-address override takes
// precedence over the service-group metadata quota; a resolved -1 skips the limit
// check. The snapshot is read before acquiring the lock. Signatures and token
// eligibility must already have been validated by the payment handler.
// The returned transaction must be finalized with Commit or Rollback.
func (h *lockingUserService) StartTransaction(payment *Payment) (transaction Transaction, err error) {

	userKey, err := h.UserKey(payment)
	if err != nil {
		return nil, errs.NewPaymentError(errs.Internal, "payment freeCallUserKey error: %s", err.Error())
	}

	freeCallUserData, ok, err := h.User(userKey)
	if err != nil {
		return nil, errs.NewPaymentError(errs.Internal, "payment freeCallUserData error: %s", err.Error())
	}

	if !ok {
		zap.L().Warn("Payment freeCallUserData not found")
		return nil, errs.NewPaymentError(errs.Unauthenticated, "payment freeCallUserData \"%v\" not found", userKey)
	}

	freeCallUserData.ServiceId = userKey.ServiceId
	freeCallUserData.OrganizationId = userKey.OrganizationId
	freeCallUserData.GroupID = userKey.GroupID
	freeCallUserData.Address = userKey.Address

	lock, ok, err := h.locker.Lock(userKey.String())
	if err != nil {
		return nil, errs.NewPaymentError(errs.Internal, "cannot get mutex for user: %v", userKey)
	}
	if !ok {
		return nil, errs.NewPaymentError(errs.FailedPrecondition, "another transaction on this user: %v is in progress", userKey)
	}
	defer func(lock storage.Lock) {
		if err != nil {
			e := lock.Unlock()
			if e != nil {
				// TODO: Notify the operator when a user lock cannot be released.
				zap.L().Error("Transaction is cancelled because of err, but freeCallUserData cannot be unlocked. All other transactions on this freeCallUserData will be blocked until unlock. Please unlock freeCallUserData manually.",
					zap.Any("userKey", userKey),
					zap.Error(err))
			}
		}
	}(lock)

	// A zero override selects the service group's JSON metadata value.
	// Registry stores its metadata URI; it does not validate this quota.
	allowed := config.GetFreeCallsAllowed(userKey.Address)
	if allowed == 0 {
		allowed = h.serviceMetadata.GetFreeCallsAllowed()
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

	return &userTransaction{
		payment:         *payment,
		freeCallUserKey: userKey,
		freeCallUser:    freeCallUserData,
		lock:            lock,
		service:         h,
	}, nil
}

func (transaction *userTransaction) Commit() error {
	defer func(payment *userTransaction) {
		err := payment.lock.Unlock()
		if err != nil {
			// TODO: Notify the operator when a user lock cannot be released.
			zap.L().Error("free call user cannot be unlocked because of error."+
				" All other transactions on this channel will be blocked until unlock."+
				" Please unlock user for free calls manually.", zap.Any("transaction", payment), zap.Error(err))
		} else {
			zap.L().Debug("free call user unlocked")
		}
	}(transaction)

	IncrementCallCount(transaction.User())
	err := transaction.service.storage.Put(
		transaction.freeCallUserKey,
		transaction.User(),
	)
	if err != nil {
		zap.L().Error("Unable to store new transaction free call user state")
		return errs.NewPaymentError(errs.Internal, "unable to store new transaction free call user state")
	}

	zap.L().Debug("Free Call Payment completed")
	return nil
}

func (transaction *userTransaction) Rollback() error {
	defer func(payment *userTransaction) {
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
