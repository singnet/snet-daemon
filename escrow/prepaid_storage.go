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

//Used when the Request comes in
func (payment *PrePaidPayment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v}", payment.ChannelID, payment.OrganizationId, payment.GroupId)
}

//Kept the bare minimum here , other details like group and sender address of this channel can
//always be retrieved from BlockChain
type PrePaidData struct {
	Amount *big.Int
}

type PrePaidDataKey struct {
	ChannelID *big.Int
	UsageType string
}

func (data *PrePaidData) String() string {
	return fmt.Sprintf("{Amount:%v}", data.Amount)
}
func (key *PrePaidDataKey) String() string {
	return fmt.Sprintf("{ChannelID:%v,UsageType:%v}", key.ChannelID, key.UsageType)
}

const (
	USED_AMOUNT    string = "U"
	PLANNED_AMOUNT string = "P"
	REFUND_AMOUNT  string = "R"
)

//This will ony be used for doing any business checks
type PrePaidUsageData struct {
	ChannelID       *big.Int
	PlannedAmount   *big.Int
	UsedAmount      *big.Int
	RefundAmount    *big.Int
	UpdateUsageType string
}

func (data *PrePaidUsageData) String() string {
	return fmt.Sprintf("{ChannelID:%v,PlannedAmount:%v,UsedAmount:%v,RefundAmount:%v,UsageTpe:%v}",
		data.ChannelID, data.PlannedAmount, data.UsedAmount, data.RefundAmount, data.UpdateUsageType)
}

func (data *PrePaidUsageData) GetAmountForUsageType() (*big.Int, error) {
	switch data.UpdateUsageType {
	case PLANNED_AMOUNT:
		return data.PlannedAmount, nil
	case REFUND_AMOUNT:
		return data.RefundAmount, nil
	case USED_AMOUNT:
		return data.UsedAmount, nil
	}
	return nil, fmt.Errorf("Unknown Usage Type %v", data.UpdateUsageType)
}

func (data PrePaidUsageData) Clone() *PrePaidUsageData {
	return &PrePaidUsageData{
		ChannelID:       data.ChannelID,
		PlannedAmount:   big.NewInt(0).Set(data.PlannedAmount),
		UsedAmount:      big.NewInt(0).Set(data.UsedAmount),
		RefundAmount:    big.NewInt(0).Set(data.RefundAmount),
		UpdateUsageType: data.UpdateUsageType,
	}
}

// NewPrepaidStorage returns new instance of TypedAtomicStorage
func NewPrepaidStorage(atomicStorage AtomicStorage) TypedAtomicStorage {
	return &TypedAtomicStorageImpl{
		atomicStorage:     NewPrefixedAtomicStorage(atomicStorage, "/PrePaid/storage"),
		keySerializer:     serialize,
		keyDeserializer:   deserialize,
		keyType:           reflect.TypeOf(PrePaidDataKey{}),
		valueSerializer:   serialize,
		valueDeserializer: deserialize,
		valueType:         reflect.TypeOf(PrePaidData{}),
	}
}
