package freecall

import (
	"fmt"
	"math/big"
)

// Payment contains the signed credentials for a free call.
type Payment struct {
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

func (key *Payment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v}", key.Address, key.UserID, key.OrganizationId,
		key.ServiceId)
}

// UserKey identifies a user's allowance within a service and replica group.
type UserKey struct {
	UserId         string
	Address        string
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (key *UserKey) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v/%v}", key.Address, key.UserId, key.OrganizationId,
		key.ServiceId, key.GroupID)
}

// UserData is persisted using gob. Field names are kept for compatibility with
// records written before free-call support was moved out of escrow.
type UserData struct {
	Address        string
	UserID         string
	FreeCallsMade  int
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (data *UserData) String() string {
	return fmt.Sprintf("{Addr:%v (id:%v) has made %v free calls for org_id=%v, service_id=%v, group_id=%v }", data.Address, data.UserID,
		data.FreeCallsMade, data.OrganizationId, data.ServiceId, data.GroupID)
}

// UserService tracks usage and starts transactions against a user's allowance.
type UserService interface {
	UserKey(payment *Payment) (userKey *UserKey, err error)

	// User returns a new entry if the user has no stored usage yet.
	User(key *UserKey) (freeCallUser *UserData, ok bool, err error)

	ListUsers() (freeCallUsers []*UserData, err error)

	StartTransaction(payment *Payment) (transaction Transaction, err error)
}

// Transaction holds the user lock until Commit or Rollback completes.
type Transaction interface {
	User() *UserData

	Commit() error

	Rollback() error
}

// IncrementCallCount records one successful free call.
func IncrementCallCount(user *UserData) {
	user.FreeCallsMade++
}
