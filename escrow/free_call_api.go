package escrow

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"

	"github.com/singnet/snet-daemon/blockchain"
)

func FreeCallID(userAddress *common.Address, serviceId string) string {
	return fmt.Sprintf("%v/%v", userAddress, serviceId)
}

type FreeCallUserKey struct {
	userId         string
	organizationId string
	serviceId      string
	groupID        [32]byte
}

func (key *FreeCallUserKey) String() string {
	return fmt.Sprintf("{ID: %v/%v/%v/%v}", key.userId, key.organizationId,
		key.serviceId, blockchain.BytesToBase64(key.groupID[:]))
}

type FreeCallUserData struct {
	UserId        string
	ServiceId     string
	OrgId         string
	GroupID       [32]byte
	FreeCallsMade int
}

func (data *FreeCallUserData) String() string {
	return fmt.Sprintf("{UserId: %v, OrgId: %v, ServiceId: %v, , groupID: %v",
		data.UserId, data.ServiceId,
		data.OrgId, blockchain.BytesToBase64(data.GroupID[:]))
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
