package escrow

import (
	"fmt"
	"math/big"
)

type PrePaidService interface {
	GetPrePaidUserKey(payment *PrePaidPayment) (userKey *PrePaidChannelKey, err error)

	GetPrePaidUser(key *PrePaidChannelKey) (PrePaidUser *PrePaidUsageData, ok bool, err error)

	ListPrePaidUsers() (PrePaidUsers []*PrePaidUsageData, err error)

	UpdateUsage(key *PrePaidChannelKey, usage UpdateUsage, revisedAmount *big.Int) error
}

type PrePaidTransaction interface {
	PrePaidKey() *PrePaidChannelKey
	Price() *big.Int
	Commit() error
	Rollback() error
}

type prePaidTransactionImpl struct {
	key   *PrePaidChannelKey
	price *big.Int
}

func (transaction prePaidTransactionImpl) PrePaidKey() *PrePaidChannelKey {
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

//First param is the used Amount, Second param is the Planned Amount, Planned amount will be updated only when
//the second param is passed
type UpdateUsage func(old interface{}, params ...interface{}) (err error)

var (
	UpdateUsedAmount = func(old interface{}, params ...interface{}) (err error) {
		oldValue := old.(*PrePaidUsageData)
		newValue := oldValue.Clone()
		if len(params) == 0 {
			return fmt.Errorf("You need to specify atleast the Usage amount to be revised")
		}
		usage := params[0].(*big.Int)
		//Check if planned amount < used amount , error out
		if newValue.PlannedAmount.Cmp(newValue.UsedAmount.Add(newValue.UsedAmount, usage)) < 0 {
			return fmt.Errorf("Current Usage:%v + Price:%v > Planned Usage%v",
				oldValue.UsedAmount, params, oldValue.PlannedAmount)
		}
		//This can happen when Usage is Negative ( we had incremented the usage, but now need to reduce as the
		//Service call Failed, this is to check make sure we dont go negative
		if newValue.UsedAmount.Int64() < 0 {
			newValue.UsedAmount = big.NewInt(0)
		}
		if len(params) == 1 {
			return nil
		}
		//if we have 2 params , the second param is the revised Planned Amount
		revisedPlannedAmount := params[1].(*big.Int)
		if oldValue.PlannedAmount.Cmp(revisedPlannedAmount) >= 0 {
			return fmt.Errorf("Current Planned Amount:%v > Revised Planned Amount:%v",
				oldValue.PlannedAmount, revisedPlannedAmount)
		}

		newValue.PlannedAmount = revisedPlannedAmount

		return nil
	}
)
