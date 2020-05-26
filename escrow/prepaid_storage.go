package escrow

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
)

// To Support PrePaid calls and also concurrency
type PrePaidPayment struct {
	ChannelID      *big.Int
	OrganizationId string
	GroupId        string
	AuthToken      []byte
}

func (payment *PrePaidPayment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v}", payment.ChannelID, payment.OrganizationId, payment.GroupId)
}

func (payment *PrePaidPayment) GetKey() *PrePaidUserKey {
	return &PrePaidUserKey{ChannelID: payment.ChannelID}
}

type PrePaidUserKey struct {
	ChannelID *big.Int
}

func (key *PrePaidUserKey) String() string {
	return fmt.Sprintf("{ID:%v}", key.ChannelID)
}

func (p *PrePaidUserKey) ID() string {
	return p.String()
}

type PrePaidUsageData struct {
	SenderAddress  common.Address
	ChannelID      *big.Int
	PlannedAmount  *big.Int
	UsedAmount     *big.Int
	OrganizationId string
	GroupID        string
}

func (data PrePaidUsageData) Clone() *PrePaidUsageData {
	return &PrePaidUsageData{
		SenderAddress:  data.SenderAddress,
		ChannelID:      data.ChannelID,
		OrganizationId: data.OrganizationId,
		GroupID:        data.GroupID,
		PlannedAmount:  big.NewInt(data.PlannedAmount.Int64()),
		UsedAmount:     big.NewInt(data.UsedAmount.Int64()),
	}
}

func (oldValue *PrePaidUsageData) Validate(newValue *PrePaidUsageData) error {
	if oldValue.PlannedAmount.Cmp(newValue.PlannedAmount) > 0 {
		return fmt.Errorf("new Planned amount:%v cannot be less "+
			"than the amount already planned:%v", newValue.PlannedAmount, oldValue.PlannedAmount)
	}
	return nil

}

func (data *PrePaidUsageData) String() string {
	return fmt.Sprintf("{User %v on Channel %v has planned amount:%v, used amount:%v "+
		"for the organization_id:%v and group_id=%v }", data.SenderAddress,
		data.ChannelID, data.PlannedAmount, data.UsedAmount, data.OrganizationId, data.GroupID)
}

// PrepaidStorage is a storage for PrepaidChannelData by
// PrepaidChannelKey based on TypedAtomicStorage implementation
type PrepaidStorage struct {
	delegate TypedAtomicStorage
}

// NewPrepaidStorage returns new instance of PrepaidStorage
// implementation
func NewPrepaidStorage(atomicStorage AtomicStorage) *PrepaidStorage {
	return &PrepaidStorage{
		delegate: &TypedAtomicStorageImpl{

			atomicStorage:     NewPrefixedAtomicStorage(atomicStorage, "/PrePaid/storage"),
			keySerializer:     serialize,
			valueSerializer:   serialize,
			valueDeserializer: deserialize,
			valueType:         reflect.TypeOf(PrePaidUsageData{}),
		},
	}
}

func (storage *PrepaidStorage) GetAll() (states []*PrePaidUsageData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*PrePaidUsageData), nil
}

func (storage *PrepaidStorage) Get(key string) (Prepaid *PrePaidUsageData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil {
		return
	}
	if !ok {
		return
	}
	return value.(*PrePaidUsageData), true, nil
}

func (storage *PrepaidStorage) Put(key *PrePaidUserKey, data PrePaidUsageData) (err error) {
	return storage.delegate.Put(key.ID(), data)
}

func (storage *PrepaidStorage) Delete(Prepaid *PrePaidUserKey) (err error) {
	return storage.delegate.Delete(Prepaid.ID())
}

func (storage *PrepaidStorage) CompareAndSwap(Prepaid *PrePaidUserKey, oldValue *PrePaidUsageData,
	newValue *PrePaidUsageData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(Prepaid.ID(), oldValue, newValue)
}
