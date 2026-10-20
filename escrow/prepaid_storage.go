package escrow

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/singnet/snet-daemon/v6/storage"
)

// PrePaidPayment contains the channel, service-group context, and JWT presented
// with a prepaid call. It does not carry a new claim signature for each call.
type PrePaidPayment struct {
	ChannelID      *big.Int
	OrganizationId string
	GroupId        string
	AuthToken      string
}

// String identifies the channel and service group without exposing AuthToken.
func (payment *PrePaidPayment) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v}", payment.ChannelID, payment.OrganizationId, payment.GroupId)
}

// PrePaidData stores only one cumulative prepaid accounting amount in token base
// units. Its channel ID and UsageType are stored separately in PrePaidDataKey.
type PrePaidData struct {
	Amount *big.Int
}

// PrePaidDataKey identifies a planned, used, or refund counter by channel ID and
// UsageType. These counters span channel nonces; nonce is not part of the key.
type PrePaidDataKey struct {
	ChannelID *big.Int
	UsageType string
}

func (data *PrePaidData) String() string {
	return fmt.Sprintf("{Amount:%v}", data.Amount)
}
func (key *PrePaidDataKey) String() string {
	return fmt.Sprintf("{ID:%v/%v}", key.ChannelID, key.UsageType)
}

const (
	// USED_AMOUNT counts charges recorded before forwarding prepaid calls,
	// including failed calls that are credited separately in REFUND_AMOUNT.
	USED_AMOUNT string = "U"
	// PLANNED_AMOUNT accumulates additional amounts authorized for prepaid use.
	PLANNED_AMOUNT string = "P"
	// REFUND_AMOUNT accumulates credits for failed calls. These credits restore
	// spending capacity without reducing USED_AMOUNT or transferring tokens.
	REFUND_AMOUNT string = "R"
)

// PrePaidUsageData combines the three stored counters for an atomic usage update.
// UpdateUsageType selects the counter written back after validation.
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

// Clone copies the accounting amounts so changes do not affect the source
// counters. ChannelID is shared and must not be mutated through the clone.
func (data PrePaidUsageData) Clone() *PrePaidUsageData {
	return &PrePaidUsageData{
		ChannelID:       data.ChannelID,
		PlannedAmount:   big.NewInt(0).Set(data.PlannedAmount),
		UsedAmount:      big.NewInt(0).Set(data.UsedAmount),
		RefundAmount:    big.NewInt(0).Set(data.RefundAmount),
		UpdateUsageType: data.UpdateUsageType,
	}
}

func serializePrePaidKey(key any) (serialized string, err error) {
	myKey := key.(PrePaidDataKey)
	return myKey.String(), nil
}

// NewPrepaidStorage stores PrePaidData values under channel/category keys in the
// /PrePaid/storage namespace of the supplied atomic storage backend.
func NewPrepaidStorage(atomicStorage storage.AtomicStorage) storage.TypedAtomicStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/PrePaid/storage")
	storage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializePrePaidKey, reflect.TypeFor[PrePaidDataKey](), storage.EncodeGob, storage.DecodeGob,
		reflect.TypeFor[PrePaidData](),
	)
	return storage
}
