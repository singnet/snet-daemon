package escrow

import (
	"fmt"
	"math/big"
)

// To Support Free calls
type FreeCallPayment struct {
	// Has the ID of the user making the call
	UserId string
	// Service ID
	ServiceId string
	// Organization Id
	OrganizationId string
	// Current block number
	CurrentBlockNumber *big.Int
	// Signature passed
	Signature []byte
	// Group ID
	GroupId string
	// Auth Token Passed
	AuthToken []byte
	// Token expiration date in blocks
	AuthTokenExpiryBlockNumber *big.Int
}

func (key *FreeCallPayment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v}", key.UserId, key.OrganizationId,
		key.ServiceId)
}

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
	UserId         string
	FreeCallsMade  int
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (data *FreeCallUserData) String() string {
	return fmt.Sprintf("{User %v has made %v free calls for org_id=%v, service_id=%v, group_id=%v }", data.UserId,
		data.FreeCallsMade, data.OrganizationId, data.ServiceId, data.GroupID)
}

type FreeCallUserService interface {
	GetFreeCallUserKey(payment *FreeCallPayment) (userKey *FreeCallUserKey, err error)

	//if the user details are not found, send back a new entry to be persisted
	FreeCallUser(key *FreeCallUserKey) (freeCallUser *FreeCallUserData, ok bool, err error)

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
