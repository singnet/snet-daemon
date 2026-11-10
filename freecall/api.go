package freecall

import (
	"fmt"
	"math/big"
)

// Payment contains caller-signed credentials and a provider-issued free-call token.
// It carries no MPE channel ID, nonce, payment amount, or claim authorization.
type Payment struct {
	// Address is the caller's signing address. Wallet callers use their own
	// address; trusted backends may identify their end user through UserID.
	// Caller-signed messages preserve this exact address text.
	Address string

	// UserID is the optional identity bound to the token. Token issuance permits
	// a nonempty ID only for addresses in trusted_free_call_signers.
	UserID string
	// ServiceId and OrganizationId identify usage records. Request signatures
	// use the corresponding daemon configuration values.
	ServiceId      string
	OrganizationId string
	// CurrentBlockNumber supplies request freshness independently of token expiry.
	CurrentBlockNumber *big.Int

	// Signature authenticates the caller's request, including the full AuthToken,
	// organization/service/group identifiers, and current block.
	Signature []byte

	// GroupId is the metadata group ID string used in the caller's signature,
	// not the daemon group name used in the provider's token signature.
	GroupId string

	// AuthToken is the full raw token: provider signature || "_" || decimal deadline.
	AuthToken []byte

	// AuthTokenParsed is the provider signature extracted by ParseToken, without
	// the underscore and expiration suffix.
	AuthTokenParsed []byte

	// AuthTokenExpiryBlockNumber is the inclusive expiration block from AuthToken.
	AuthTokenExpiryBlockNumber *big.Int
}

func (key *Payment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v/%v}", key.Address, key.UserID, key.OrganizationId,
		key.ServiceId)
}

// UserKey identifies usage by caller address, optional user ID, organization,
// service, and replica group. Address and user-ID text are not normalized in
// storage, although per-address quota configuration uses case-insensitive lookup.
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

// UserData stores a usage counter for one UserKey, not an on-chain MPE balance.
// It is persisted using gob. Field names are kept for compatibility with records
// written before free-call support was moved out of escrow.
type UserData struct {
	Address string
	UserID  string
	// FreeCallsMade counts calls committed to this usage record.
	FreeCallsMade  int
	OrganizationId string
	ServiceId      string
	GroupID        string
}

func (data *UserData) String() string {
	return fmt.Sprintf("{Addr:%v (id:%v) has made %v free calls for org_id=%v, service_id=%v, group_id=%v }", data.Address, data.UserID,
		data.FreeCallsMade, data.OrganizationId, data.ServiceId, data.GroupID)
}

// UserService tracks usage and checks allowances after the caller's signatures
// have been authenticated. It does not read or update MPE channel state.
type UserService interface {
	// UserKey uses payment identity fields and the service's replica-group
	// reader; it does not take the storage group from payment.GroupId.
	UserKey(payment *Payment) (userKey *UserKey, err error)

	// User returns stored usage or an unpersisted zero-count entry with ok true
	// when the key is absent. Reading a new user does not consume an allowance.
	User(key *UserKey) (freeCallUser *UserData, ok bool, err error)

	// ListUsers returns persisted usage records under the configured storage scope.
	ListUsers() (freeCallUsers []*UserData, err error)

	// StartTransaction checks the usage snapshot against the configured quota
	// and returns a handle holding the user lock until Commit or Rollback.
	// Payment authentication is the caller's responsibility.
	StartTransaction(payment *Payment) (transaction Transaction, err error)
}

// Transaction holds a user lock that Commit or Rollback attempts to release.
// Call exactly one completion method. Unlock failures are logged, not returned.
type Transaction interface {
	// User returns the usage snapshot associated with this transaction.
	User() *UserData

	// Commit increments and persists the usage counter after a successful call.
	// It attempts to unlock even if writing the counter fails.
	Commit() error

	// Rollback attempts to unlock without persisting an increment.
	Rollback() error
}

// IncrementCallCount increments the in-memory counter; Commit persists the change.
func IncrementCallCount(user *UserData) {
	user.FreeCallsMade++
}
