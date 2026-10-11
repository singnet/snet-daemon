package escrow

import (
	"errors"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
)

type transactionStorage struct {
	storage.TypedAtomicStorage
	apply func(storage.TypedCASRequest) (bool, error)
}

func (s transactionStorage) ExecuteTransaction(request storage.TypedCASRequest) (bool, error) {
	return s.apply(request)
}

func TestUpdateUsageRejectsInvalidType(t *testing.T) {
	service := NewPrePaidService(nil, nil, nil)
	require.ErrorContains(t, service.UpdateUsage(big.NewInt(1), big.NewInt(1), "invalid"), "Unknown Update type")
}

func TestUpdateUsageTransactionFailures(t *testing.T) {
	for _, tt := range []struct {
		name    string
		apply   func(storage.TypedCASRequest) (bool, error)
		wantErr string
	}{
		{"storage", func(storage.TypedCASRequest) (bool, error) { return false, errors.New("storage unavailable") }, "storage unavailable"},
		{"conflict", func(storage.TypedCASRequest) (bool, error) { return false, nil }, "Error in executing ExecuteTransaction"},
		{"limit exceeded", func(request storage.TypedCASRequest) (bool, error) {
			_, ok, err := request.Update(nil)
			return ok, err
		}, "Usage Exceeded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPrePaidService(transactionStorage{apply: tt.apply}, nil, nil)
			require.ErrorContains(t, service.UpdateUsage(big.NewInt(1), big.NewInt(1), USED_AMOUNT), tt.wantErr)
		})
	}
}

func TestUsageUpdatesRejectUnknownStoredType(t *testing.T) {
	data := []storage.TypedKeyValueData{{Key: PrePaidDataKey{ChannelID: big.NewInt(1), UsageType: "invalid"}, Value: &PrePaidData{Amount: big.NewInt(1)}, Present: true}}
	for _, tt := range []struct {
		name   string
		update ConditionFunc
	}{
		{"used", IncrementUsedAmount}, {"planned", IncrementPlannedAmount}, {"refund", IncrementRefundAmount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values, err := tt.update(data, big.NewInt(1), big.NewInt(1))
			require.Nil(t, values)
			require.ErrorContains(t, err, "Unknown Usage Type")
		})
	}
}
