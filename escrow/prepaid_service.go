package escrow

import (
	"math/big"
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

func (h *lockingPrepaidService) GetPrePaidUserKey(payment *PrePaidPayment) (userKey *PrePaidChannelKey, err error) {
	return nil, nil
}

func (h *lockingPrepaidService) GetPrePaidUser(key *PrePaidChannelKey) (PrePaidUser *PrePaidUsageData, ok bool, err error) {
	return h.storage.Get(key.ID())
}

func (h *lockingPrepaidService) UpdateUsage(key *PrePaidChannelKey,
	usage UpdateUsage, revisedAmount *big.Int) (err error) {
	//todo
	return nil
}
