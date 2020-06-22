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
type PrePaidDataUnit struct {
	ChannelID *big.Int
	Amount    *big.Int
	UsageType string
}

func (data *PrePaidDataUnit) String() string {
	return fmt.Sprintf("{ChannelID:%v,Amount:%v,UsageType:%v}", data.ChannelID, data.Amount, data.UsageType)
}

func (data *PrePaidDataUnit) Key() string {
	return fmt.Sprintf("%v/%v", data.ChannelID, data.UsageType)
}

const (
	USED_AMOUNT    string = "U"
	PLANNED_AMOUNT string = "P"
	REFUND_AMOUNT  string = "R"
)

//This will ony be used for doing any business checks
type PrePaidUsageData struct {
	SenderAddress   string
	ChannelID       *big.Int
	PlannedAmount   *big.Int
	UsedAmount      *big.Int
	FailedAmount    *big.Int
	OrganizationId  string
	GroupID         string
	UpdateUsageType string
}

func (data *PrePaidUsageData) String() string {
	return fmt.Sprintf("{SenderAddress:%v,ChannelID:%v,PlannedAmount:%v,UsedAmount:%v,"+
		"OrganizationId:%v,GroupID:%v}", data.SenderAddress,
		data.ChannelID, data.PlannedAmount, data.UsedAmount, data.OrganizationId, data.GroupID)
}

func (data *PrePaidUsageData) GetAmountForUsageType() (*big.Int, error) {
	switch data.UpdateUsageType {
	case PLANNED_AMOUNT:
		return data.PlannedAmount, nil
	case REFUND_AMOUNT:
		return data.FailedAmount, nil
	case USED_AMOUNT:
		return data.UsedAmount, nil
	}
	return nil, fmt.Errorf("Unknown Usage Type %v", data.UpdateUsageType)
}

func (data PrePaidUsageData) Clone() *PrePaidUsageData {
	return &PrePaidUsageData{
		SenderAddress:   data.SenderAddress,
		ChannelID:       data.ChannelID,
		OrganizationId:  data.OrganizationId,
		GroupID:         data.GroupID,
		PlannedAmount:   big.NewInt(data.PlannedAmount.Int64()),
		UsedAmount:      big.NewInt(data.UsedAmount.Int64()),
		FailedAmount:    big.NewInt(data.FailedAmount.Int64()),
		UpdateUsageType: data.UpdateUsageType,
	}
}

// NewPrepaidStorage returns new instance of TypedAtomicStorage
func NewPrepaidStorage(atomicStorage AtomicStorage) TypedAtomicStorage {
	return &TypedAtomicStorageImpl{
		atomicStorage:     NewPrefixedAtomicStorage(atomicStorage, "/PrePaid/storage"),
		keySerializer:     serialize,
		valueSerializer:   serialize,
		valueDeserializer: deserialize,
		valueType:         reflect.TypeOf(PrePaidDataUnit{}),
	}
}
