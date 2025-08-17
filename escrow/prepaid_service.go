package escrow

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/singnet/snet-daemon/v6/storage"
)

type lockingPrepaidService struct {
	storage        storage.TypedAtomicStorage
	validator      *PrePaidPaymentValidator
	replicaGroupID func() ([32]byte, error)
}

func NewPrePaidService(
	storage storage.TypedAtomicStorage,
	prepaidValidator *PrePaidPaymentValidator, groupIdReader func() ([32]byte, error)) PrePaidService {
	return &lockingPrepaidService{
		storage:        storage,
		validator:      prepaidValidator,
		replicaGroupID: groupIdReader,
	}
}

func (h *lockingPrepaidService) GetUsage(key PrePaidDataKey) (data *PrePaidData, ok bool, err error) {

	value, ok, err := h.storage.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*PrePaidData), ok, err
}

// ConditionFunc Defines the condition that needs to be met, it generates the respective typed Data when
// conditions are satisfied; you define your own validations in here
// It takes in the latest typed values read.
type ConditionFunc func(conditionValues []storage.TypedKeyValueData, revisedAmount *big.Int, channelId *big.Int) ([]storage.TypedKeyValueData, error)

func (h *lockingPrepaidService) UpdateUsage(channelId *big.Int, revisedAmount *big.Int, updateUsageType string) (err error) {
	var conditionFunc ConditionFunc = nil

	switch updateUsageType {
	case USED_AMOUNT:
		conditionFunc = IncrementUsedAmount
	case PLANNED_AMOUNT:
		conditionFunc = IncrementPlannedAmount
	case REFUND_AMOUNT:
		conditionFunc = IncrementRefundAmount
	default:
		return fmt.Errorf("Unknown Update type %v", updateUsageType)
	}

	typedUpdateFunc := func(conditionValues []storage.TypedKeyValueData) (update []storage.TypedKeyValueData, ok bool, err error) {
		var newValues []storage.TypedKeyValueData
		if newValues, err = conditionFunc(conditionValues, revisedAmount, channelId); err != nil {
			return nil, false, err
		}
		return newValues, true, nil
	}
	typedKeys := getAllKeys(channelId)
	request := storage.TypedCASRequest{
		Update:                  typedUpdateFunc,
		RetryTillSuccessOrError: true,
		ConditionKeys:           typedKeys,
	}
	ok, err := h.storage.ExecuteTransaction(request)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Error in executing ExecuteTransaction for usage type"+
			"  %v on channel %v ", updateUsageType, channelId)
	}
	return nil
}

func getAllKeys(channelId *big.Int) []any {
	keys := make([]any, 3)
	for i, usageType := range []string{REFUND_AMOUNT, PLANNED_AMOUNT, USED_AMOUNT} {
		keys[i] = PrePaidDataKey{ChannelID: channelId, UsageType: usageType}
	}
	return keys
}

// this function will be used to read typed data ,convert it in to a business structure
// on which validations can be easily performed and return back the business structure.
func convertTypedDataToPrePaidUsage(data []storage.TypedKeyValueData) (new *PrePaidUsageData, err error) {
	usageData := &PrePaidUsageData{PlannedAmount: big.NewInt(0),
		UsedAmount: big.NewInt(0), RefundAmount: big.NewInt(0)}
	for _, usageType := range data {
		key := usageType.Key.(PrePaidDataKey)
		usageData.ChannelID = key.ChannelID
		if !usageType.Present {
			continue
		}
		data := usageType.Value.(*PrePaidData)
		if strings.Compare(key.UsageType, USED_AMOUNT) == 0 {
			usageData.UsedAmount = data.Amount
		} else if strings.Compare(key.UsageType, PLANNED_AMOUNT) == 0 {
			usageData.PlannedAmount = data.Amount
		} else if strings.Compare(key.UsageType, REFUND_AMOUNT) == 0 {
			usageData.RefundAmount = data.Amount
		} else {
			return nil, fmt.Errorf("Unknown Usage Type %v", key.UsageType)
		}
	}
	return usageData, nil
}

func BuildOldAndNewValuesForCAS(data *PrePaidUsageData) (newValues []storage.TypedKeyValueData, err error) {
	updateUsageData := &PrePaidData{}
	updateUsageKey := PrePaidDataKey{ChannelID: data.ChannelID, UsageType: data.UpdateUsageType}
	if amt, err := data.GetAmountForUsageType(); err != nil {
		return nil, err
	} else {
		updateUsageData.Amount = amt
	}
	newValue := storage.TypedKeyValueData{Key: updateUsageKey, Value: updateUsageData, Present: true}
	newValues = make([]storage.TypedKeyValueData, 1)
	newValues[0] = newValue

	return newValues, nil
}

var (
	IncrementUsedAmount ConditionFunc = func(conditionValues []storage.TypedKeyValueData, revisedAmount *big.Int, channelId *big.Int) (newValues []storage.TypedKeyValueData, err error) {
		oldState, err := convertTypedDataToPrePaidUsage(conditionValues)
		if err != nil {
			return nil, err
		}
		oldState.ChannelID = channelId
		newState := oldState.Clone()
		usageKey := PrePaidDataKey{UsageType: USED_AMOUNT, ChannelID: oldState.ChannelID}
		updateDetails(newState, usageKey, revisedAmount)
		if newState.UsedAmount.Cmp(oldState.PlannedAmount.Add(oldState.PlannedAmount, oldState.RefundAmount)) > 0 {
			return nil, fmt.Errorf("Usage Exceeded on channel Id %v", oldState.ChannelID)
		}
		return BuildOldAndNewValuesForCAS(newState)

	}
	//Make sure you update the planned amount ONLY when the new value is greater than what was last persisted
	IncrementPlannedAmount ConditionFunc = func(conditionValues []storage.TypedKeyValueData, revisedAmount *big.Int, channelId *big.Int) (newValues []storage.TypedKeyValueData, err error) {
		oldState, err := convertTypedDataToPrePaidUsage(conditionValues)
		if err != nil {
			return nil, err
		}
		//Assuming there are no entries yet on this channel, it is very easy to pass the channel ID to the condition
		//function and pick it from there
		oldState.ChannelID = channelId
		newState := oldState.Clone()
		usageKey := PrePaidDataKey{UsageType: PLANNED_AMOUNT, ChannelID: oldState.ChannelID}
		updateDetails(newState, usageKey, revisedAmount)
		return BuildOldAndNewValuesForCAS(newState)

	}
	//If there is no refund amount yet, put it , else add latest value in DB with the additional refund to be done
	IncrementRefundAmount ConditionFunc = func(conditionValues []storage.TypedKeyValueData, revisedAmount *big.Int, channelId *big.Int) (newValues []storage.TypedKeyValueData, err error) {
		newState, err := convertTypedDataToPrePaidUsage(conditionValues)
		if err != nil {
			return nil, err
		}
		newState.ChannelID = channelId
		usageKey := PrePaidDataKey{UsageType: REFUND_AMOUNT, ChannelID: newState.ChannelID}
		updateDetails(newState, usageKey, revisedAmount)
		return BuildOldAndNewValuesForCAS(newState)

	}
)

func updateDetails(usageData *PrePaidUsageData, key PrePaidDataKey, usage *big.Int) {
	usageData.ChannelID = key.ChannelID
	usageData.UpdateUsageType = key.UsageType
	switch key.UsageType {
	case USED_AMOUNT:
		usageData.UsedAmount.Add(usage, usageData.UsedAmount)
	case PLANNED_AMOUNT:
		usageData.PlannedAmount.Add(usage, usageData.PlannedAmount)
	case REFUND_AMOUNT:
		usageData.RefundAmount.Add(usage, usageData.RefundAmount)
	}
}
