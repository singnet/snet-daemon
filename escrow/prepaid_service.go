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

func (h *lockingPrepaidService) GetPrePaidUserKey(payment *PrePaidPayment) (userKey *PrePaidUserKey, err error) {
	return nil, nil
}

func (h *lockingPrepaidService) GetPrePaidUser(key *PrePaidUserKey) (PrePaidUser *PrePaidUsageData, ok bool, err error) {
	return h.storage.Get(key.ID())
}

func (h *lockingPrepaidService) UpdateUsage(key *PrePaidUserKey,
	usage UpdateUsage, revisedAmount *big.Int) (err error) {
	cas := &ValidateAndUpdateStorageDetails{
		Validate: usage,
		Retry:    true,
		Key:      key,
		Params:   revisedAmount,
	}

	return h.storage.VerifyAndUpdate(cas)
}
