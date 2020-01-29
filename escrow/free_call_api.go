package escrow

import (
	"fmt"
)

type FreeCallUserKey struct {
	UserId         string
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (key *FreeCallUserKey) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v}", key.UserId, key.OrganizationId,
		key.ServiceId, key.GroupID)
}

type FreeCallUserData struct {
	FreeCallsMade int
}

func (data *FreeCallUserData) String() string {
	return fmt.Sprintf("{FreeCallsMade: %v}",
		data.FreeCallsMade)
}

type FreeCallUserService interface {
	FreeCallUserUsage(key *FreeCallUserKey) (freeCallUser *FreeCallUserData, ok bool, err error)

	ListFreeCallUsers() (freeCallUsers []*FreeCallUserData, err error)

	StartFreeCallUserTransaction(payment *FreeCallPayment) (transaction FreeCallTransaction, err error)
}

type FreeCallTransaction interface {
	FreeCallUser() *FreeCallUserData

	Commit() error

	Rollback() error
}

type FreeCallUserUpdate func(user *FreeCallUserData)

var (
	IncrementFreeCallCount FreeCallUserUpdate = func(user *FreeCallUserData) {
		user.FreeCallsMade = user.FreeCallsMade + 1
	}
)
