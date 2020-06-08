package escrow

import (
	"fmt"
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

func (payment *PrePaidPayment) GetKey() *PrePaidChannelKey {
	return &PrePaidChannelKey{ChannelID: payment.ChannelID}
}

type PrePaidChannelKey struct {
	ChannelID *big.Int
}

func (key *PrePaidChannelKey) String() string {
	return fmt.Sprintf("{ID:%v}", key.ChannelID)
}

func (p *PrePaidChannelKey) ID() string {
	return p.String()
}

type PrePaidUsageDataType string

type PrePaidDataUnit struct {
	ChannelID *big.Int
	Amount    *big.Int
	UsageType string
}

func (data *PrePaidDataUnit) String() string {
	return fmt.Sprintf("{ChannelID:%v,Amount:%v,UsgaeType:%v}", data.ChannelID, data.Amount, data.UsageType)
}

func (data *PrePaidDataUnit) Key() string {
	return fmt.Sprintf("{ID:%v/%v}", data.ChannelID, data.UsageType)
}

const (
	USED_AMOUNT    string = "U"
	PLANNED_AMOUNT string = "P"
	FAILED_AMOUNT  string = "F"
)

type PrePaidUsageData struct {
	SenderAddress       string
	ChannelID           *big.Int
	PlannedAmount       *big.Int
	UsedAmount          *big.Int
	FailedAmount        *big.Int
	OrganizationId      string
	GroupID             string
	LastModifiedVersion int64
	UsageType           string
}

func (data PrePaidUsageData) Key() string {
	dataUnit := PrePaidDataUnit{UsageType: data.UsageType}
	return dataUnit.Key()
}
func (data *PrePaidUsageData) String() string {
	return fmt.Sprintf("{\"SenderAddress\":\"%v\",\"ChannelID\":%v,\"PlannedAmount\":%v,\"UsedAmount\":%v,"+
		"\"OrganizationId\":\"%v\",\"GroupID\":\"%v\"}", data.SenderAddress,
		data.ChannelID, data.PlannedAmount, data.UsedAmount, data.OrganizationId, data.GroupID)
}

func (data PrePaidUsageData) Clone() *PrePaidUsageData {
	return &PrePaidUsageData{
		SenderAddress:  data.SenderAddress,
		ChannelID:      data.ChannelID,
		OrganizationId: data.OrganizationId,
		GroupID:        data.GroupID,
		PlannedAmount:  big.NewInt(data.PlannedAmount.Int64()),
		UsedAmount:     big.NewInt(data.UsedAmount.Int64()),
		FailedAmount:   big.NewInt(data.FailedAmount.Int64()),
	}
}

func (oldValue *PrePaidUsageData) Validate(newValue *PrePaidUsageData) error {
	if oldValue.PlannedAmount.Cmp(newValue.PlannedAmount) > 0 {
		return fmt.Errorf("new Planned amount:%v cannot be less "+
			"than the amount already planned:%v", newValue.PlannedAmount, oldValue.PlannedAmount)
	}
	return nil

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

func (storage *PrepaidStorage) UpdateUsage() (err error) {
	return nil
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

func (storage *PrepaidStorage) Put(key *PrePaidChannelKey, data PrePaidUsageData) (err error) {
	return storage.delegate.Put(key.ID(), data)
}

func (storage *PrepaidStorage) Delete(Prepaid *PrePaidChannelKey) (err error) {
	return storage.delegate.Delete(Prepaid.ID())
}

func (storage *PrepaidStorage) CompareAndSwap(Prepaid *PrePaidChannelKey, oldValue *PrePaidUsageData,
	newValue *PrePaidUsageData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(Prepaid.ID(), oldValue, newValue)
}
