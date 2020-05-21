package escrow

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// To Support PrePaid calls and also concurrency
type PrePaidPayment struct {
	ChannelID      *big.Int
	OrganizationId string
	GroupId        string
	AuthToken      []byte
}

func (key *PrePaidPayment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v}", key.ChannelID, key.OrganizationId, key.GroupId)
}

type PrePaidChannelKey struct {
	ChannelID *big.Int
}

func (key *PrePaidChannelKey) String() string {
	return fmt.Sprintf("{ID:%v}", key.ChannelID)
}

type PrePaidUserData struct {
	SenderAddress common.Address
	ChannelID     *big.Int
	PlannedAmount *big.Int
	ActualAmount  *big.Int
	//PlannedCalls and ActualCalls will come in use when you start supporting Licensing
	PlannedCalls   *big.Int
	ActualCalls    *big.Int
	OrganizationId string
	GroupID        string
}

func (data *PrePaidUserData) String() string {
	return fmt.Sprintf("{User %v on Channel %v has planned amount:%v, used amount:%v "+
		"for the organization_id:%v and group_id=%v }", data.SenderAddress,
		data.ChannelID, data.PlannedAmount, data.ActualAmount, data.OrganizationId, data.GroupID)
}

type PrePaidUserService interface {
	GetPrePaidUserKey(payment *PrePaidPayment) (userKey *PrePaidChannelKey, err error)

	PrePaidUser(key *PrePaidChannelKey) (PrePaidUser *PrePaidUserData, ok bool, err error)

	ListPrePaidUsers() (PrePaidUsers []*PrePaidUserData, err error)

	StartPrePaidUserTransaction(payment *PrePaidPayment) (transaction PrePaidTransaction, err error)
}

type PrePaidTransaction interface {
	PrePaidUser() *PrePaidUserData

	Commit() error

	Rollback() error
}

type UpdatePrePaidUsage func(user *PrePaidUserData, price *big.Int) error

var (
	UpdateUsage UpdatePrePaidUsage = func(user *PrePaidUserData, price *big.Int) error {
		user.ActualAmount = user.ActualAmount.Add(user.ActualAmount, price)
		return nil
	}
)
