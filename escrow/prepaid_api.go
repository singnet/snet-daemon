package escrow

import (
	"fmt"
	"math/big"
)

type PrePaidService interface {
	GetPrePaidUserKey(payment *PrePaidPayment) (userKey *PrePaidUserKey, err error)

	GetPrePaidUser(key *PrePaidUserKey) (PrePaidUser *PrePaidUsageData, ok bool, err error)

	ListPrePaidUsers() (PrePaidUsers []*PrePaidUsageData, err error)

	UpdateUsage(key *PrePaidUserKey, usage UpdateUsage, revisedAmount *big.Int) error
}

type PrePaidTransaction interface {
	PrePaidKey() *PrePaidUserKey
	Price() *big.Int
	Commit() error
	Rollback() error
}

type prePaidTransactionImpl struct {
	key   *PrePaidUserKey
	price *big.Int
}

func (transaction prePaidTransactionImpl) PrePaidKey() *PrePaidUserKey {
	return nil
}
func (transaction prePaidTransactionImpl) Price() *big.Int {
	return nil
}
func (transaction prePaidTransactionImpl) Commit() error {
	return nil
}
func (transaction prePaidTransactionImpl) Rollback() error {
	return nil
}

type UpdateUsage func(oldValue *PrePaidUsageData, price *big.Int) (newValue *PrePaidUsageData, err error)

var (
	IncreaseUsedAmount UpdateUsage = func(oldValue *PrePaidUsageData, price *big.Int) (newValue *PrePaidUsageData,
		err error) {
		newValue = oldValue.Clone()
		//Check if planned amount < used amount , error out
		if newValue.PlannedAmount.Cmp(newValue.UsedAmount.Add(newValue.UsedAmount, price)) < 0 {
			return nil, fmt.Errorf("Current Usage:%v + Price:%v > Planned Usage%v",
				oldValue.UsedAmount, price, oldValue.PlannedAmount)
		}
		return newValue, nil
	}
	//This will be used , when the service call errors and , you need to reduce the usage , usage is incremented
	//just before initiating the service call .
	DecreaseUsedAmount UpdateUsage = func(oldValue *PrePaidUsageData, price *big.Int) (newValue *PrePaidUsageData, err error) {
		newValue = oldValue.Clone()
		newValue.UsedAmount = newValue.UsedAmount.Sub(newValue.UsedAmount, price)
		//reset the counter to zero
		if newValue.UsedAmount.Int64() < 0 {
			newValue.UsedAmount = big.NewInt(0)
		}
		return newValue, nil
	}
	//This will be used when a request for a new Token comes in !
	IncreasePlannedUsage UpdateUsage = func(oldValue *PrePaidUsageData, revisedPlannedAmount *big.Int) (newValue *PrePaidUsageData, err error) {

		if oldValue.PlannedAmount.Cmp(revisedPlannedAmount) >= 0 {
			return nil, fmt.Errorf("Current Planned Amount:%v > Revised Planned Amount:%v",
				oldValue.PlannedAmount, revisedPlannedAmount)
		}
		newValue = oldValue.Clone()
		newValue.PlannedAmount = revisedPlannedAmount
		return newValue, nil
	}
)
