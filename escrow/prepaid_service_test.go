package escrow

import (
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	//	"github.com/stretchr/testify/assert"
	//	"github.com/stretchr/testify/suite"
)

func TestNewPrePaidService(t *testing.T) {

}

func Test_lockingPrepaidService_GetUsage(t *testing.T) {

}

func Test_lockingPrepaidService_UpdateUsage(t *testing.T) {

}

func Test_getAllKeys(t *testing.T) {
	keys := getAllKeys(big.NewInt(10))
	assert.True(t, len(keys) == 3)
	assert.Contains(t, keys, PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: USED_AMOUNT})
	assert.Contains(t, keys, PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: PLANNED_AMOUNT})
	assert.Contains(t, keys, PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: REFUND_AMOUNT})
}

func Test_convertTypedDataToPrePaidUsage(t *testing.T) {
	typedArray := make([]storage.TypedKeyValueData, 3)
	typedArray[0] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: USED_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(3)},
		Present: true,
	}
	typedArray[1] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: PLANNED_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(10)},
		Present: true,
	}

	typedArray[2] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: REFUND_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(4)},
		Present: true,
	}
	newState, err := convertTypedDataToPrePaidUsage(typedArray)
	assert.Nil(t, err)
	assert.Equal(t, newState.PlannedAmount, big.NewInt(10))
	assert.Equal(t, newState.UsedAmount, big.NewInt(3))
	assert.Equal(t, newState.RefundAmount, big.NewInt(4))

	typedArray[0] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: big.NewInt(10), UsageType: "BAD"},
		Value:   &PrePaidData{Amount: big.NewInt(3)},
		Present: true,
	}
	newState, err = convertTypedDataToPrePaidUsage(typedArray)
	assert.Equal(t, err.Error(), "Unknown Usage Type BAD")

}

func TestBuildOldAndNewValuesForCAS(t *testing.T) {
	data := &PrePaidUsageData{}
	newValues, err := BuildOldAndNewValuesForCAS(data)
	assert.Equal(t, err.Error(), "Unknown Usage Type ")
	assert.Nil(t, newValues)
	data.UpdateUsageType = PLANNED_AMOUNT
	data.PlannedAmount = big.NewInt(11)
	newValues, err = BuildOldAndNewValuesForCAS(data)
	assert.Nil(t, err)
	assert.Equal(t, newValues[0].Value, &PrePaidData{Amount: big.NewInt(11)})

}

func Test_updateDetails(t *testing.T) {
	usage := &PrePaidUsageData{PlannedAmount: big.NewInt(10),
		UpdateUsageType: PLANNED_AMOUNT}
	updateDetails(usage, PrePaidDataKey{ChannelID: big.NewInt(11), UsageType: PLANNED_AMOUNT}, big.NewInt(10))
	assert.Equal(t, usage.PlannedAmount, big.NewInt(20))

}

func Test_KeySerializeAndDeserialize(t *testing.T) {
	key := PrePaidDataKey{ChannelID: big.NewInt(11), UsageType: PLANNED_AMOUNT}
	newkey, err := serializePrePaidKey(key)
	assert.Equal(t, "{ID:11/P}", newkey)
	assert.Nil(t, err)

}

func TestFuncUsedAmount(t *testing.T) {
	channelId := big.NewInt(10)
	typedArray := make([]storage.TypedKeyValueData, 2)
	typedArray[0] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: channelId, UsageType: USED_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(3)},
		Present: true,
	}
	typedArray[1] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: channelId, UsageType: PLANNED_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(30)},
		Present: true,
	}
	newValues, err := IncrementUsedAmount(typedArray, big.NewInt(400), channelId)
	assert.Nil(t, newValues)
	assert.Equal(t, err.Error(), "Usage Exceeded on channel Id 10")
	newValues, err = IncrementUsedAmount(typedArray, big.NewInt(4), channelId)
	assert.Nil(t, err)
	assert.Equal(t, newValues[0].Value, &PrePaidData{Amount: big.NewInt(7)})
}

func TestFuncPlannedAmount(t *testing.T) {
	channelId := big.NewInt(10)
	typedArray := make([]storage.TypedKeyValueData, 1)
	typedArray[0] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: channelId, UsageType: PLANNED_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(3)},
		Present: true,
	}

	newValues, err := IncrementPlannedAmount(typedArray, big.NewInt(4), channelId)
	assert.Nil(t, err)
	assert.Equal(t, newValues[0].Value, &PrePaidData{Amount: big.NewInt(7)})
}

func TestFuncRefundAmount(t *testing.T) {
	channelId := big.NewInt(10)
	typedArray := make([]storage.TypedKeyValueData, 1)
	typedArray[0] = storage.TypedKeyValueData{
		Key:     PrePaidDataKey{ChannelID: channelId, UsageType: REFUND_AMOUNT},
		Value:   &PrePaidData{Amount: big.NewInt(3)},
		Present: true,
	}

	newValues, err := IncrementRefundAmount(typedArray, big.NewInt(1), channelId)
	assert.Nil(t, err)
	assert.Equal(t, newValues[0].Value, &PrePaidData{Amount: big.NewInt(4)})
}
