package escrow

import (
	"fmt"
	"math/big"
)

// To Support Free calls
type FreeCallPayment struct {
	//Has the Id of the user making the call
	UserId string
	//Service Id .
	ServiceId string
	//Organization Id
	OrganizationId string
	//Current block number
	CurrentBlockNumber *big.Int
	// Signature passed
	Signature []byte
	//Group ID
	GroupId string
	//Auth Token Passed
	AuthToken []byte
	//Date on when the token was issued in block number
	AuthTokenIssueDateBlockNumber *big.Int

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
	FreeCallsMade int
}

func (data *FreeCallUserData) String() string {
	return fmt.Sprintf("{FreeCallsMade: %v}",
		data.FreeCallsMade)
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
