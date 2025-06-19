package escrow

import (
	"fmt"
	"math/big"
)

type FreeCallPayment struct {
	// Address of the user or trusted backend making the request.
	// In Web3 flow, this is the user's wallet address.
	// In Web2 (e.g., marketplace), this is the backend's signing address (must be trusted).
	Address string

	UserID             string
	ServiceId          string
	OrganizationId     string
	CurrentBlockNumber *big.Int

	// Signature passed
	Signature []byte

	GroupId string

	// Auth Token Passed
	AuthToken []byte

	// Auth token without block
	AuthTokenParsed []byte

	// Token expiration date in blocks
	AuthTokenExpiryBlockNumber *big.Int
}

func (key *FreeCallPayment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v}", key.Address, key.UserID, key.OrganizationId,
		key.ServiceId)
}

type FreeCallUserKey struct {
	UserId         string
	Address        string
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (key *FreeCallUserKey) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v/%v}", key.Address, key.UserId, key.OrganizationId,
		key.ServiceId, key.GroupID)
}

type FreeCallUserData struct {
	Address        string
	UserID         string
	FreeCallsMade  int
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (data *FreeCallUserData) String() string {
	return fmt.Sprintf("{Addr:%v (id:%v) has made %v free calls for org_id=%v, service_id=%v, group_id=%v }", data.Address, data.UserID,
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
		user.FreeCallsMade += 1
	}
)
