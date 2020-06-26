package escrow

import (
	"fmt"
	"math/big"
	"strings"
)

type lockingPrepaidService struct {
	storage        TypedAtomicStorage
	validator      *PrePaidPaymentValidator
	replicaGroupID func() ([32]byte, error)
}

func NewPrePaidService(
	storage TypedAtomicStorage,
	prepaidValidator *PrePaidPaymentValidator, groupIdReader func() ([32]byte, error)) PrePaidService {
	return &lockingPrepaidService{
		storage:        storage,
		validator:      prepaidValidator,
		replicaGroupID: groupIdReader,
	}
}

func (h *lockingPrepaidService) ListPrePaidUsers() (users []*PrePaidDataUnit, err error) {
	data, err := h.storage.GetAll()
	return data.([]*PrePaidDataUnit), err
}

//Defines the condition that needs to be met, it generates the respective typed Data when
//conditions are satisfied, you define your own validations in here
//It takes in the latest typed values read.
type ConditionFunc func(params ...interface{}) ([]*TypedKeyValueData, error)

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
		return fmt.Errorf("Unknow Update type %v", updateUsageType)
	}

	typedUpdateFunc := func(conditionValues []*TypedKeyValueData) (update []*TypedKeyValueData, err error) {
		var oldValues []interface{}
		var newValues interface{}

		oldValues = make([]interface{}, len(conditionValues))
		for i, keyValue := range conditionValues {
			oldValues[i] = keyValue.Value
		}

		if newValues, err = conditionFunc(oldValues, revisedAmount); err != nil {
			return nil, err
		}

		return BuildOldAndNewValuesForCAS(newValues)
	}

	request := TypedCASRequest{
		Update:                  typedUpdateFunc,
		RetryTillSuccessOrError: true,
		ConditionKeyPrefix:      channelId.String() + "/",
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

var (
	//this function will be used to read typed data ,convert it in to a business structure
	//on which validations can be easily performed and return back the business structure.
	convertRawDataToPrePaidUsage = func(latestReadData interface{}) (new interface{}, err error) {
		data := latestReadData.([]*PrePaidDataUnit)
		usageData := &PrePaidUsageData{PlannedAmount: big.NewInt(0),
			UsedAmount: big.NewInt(0), FailedAmount: big.NewInt(0)}
		for _, usageType := range data {
			usageData.ChannelID = usageType.ChannelID
			if strings.Compare(usageType.UsageType, USED_AMOUNT) == 0 {
				usageData.UsedAmount = usageType.Amount
			} else if strings.Compare(usageType.UsageType, PLANNED_AMOUNT) == 0 {
				usageData.PlannedAmount = usageType.Amount
			} else if strings.Compare(usageType.UsageType, REFUND_AMOUNT) == 0 {
				usageData.FailedAmount = usageType.Amount
			} else {
				return nil, fmt.Errorf("Unknown Usage Type %v", usageType.String())
			}
		}
		return usageData, nil
	}
)

func BuildOldAndNewValuesForCAS(params ...interface{}) (newValues []*TypedKeyValueData, err error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("No parameters passed for the Action function")
	}
	data := params[0].(*PrePaidUsageData)
	if data == nil {
		return nil, fmt.Errorf("Expected PrePaidUsageData in Params as the first parmeter")
	}
	updateUsage := &PrePaidDataUnit{ChannelID: data.ChannelID, UsageType: data.UpdateUsageType}
	if amt, err := data.GetAmountForUsageType(); err != nil {
		return nil, err
	} else {
		updateUsage.Amount = amt
	}
	newValue := &TypedKeyValueData{Key: updateUsage.Key(), Value: updateUsage}
	newValues = make([]*TypedKeyValueData, 0)
	newValues = append(newValues, newValue)

	return newValues, nil
}

var (
	IncrementUsedAmount ConditionFunc = func(params ...interface{}) (newValues []*TypedKeyValueData, err error) {
		data := params[0].([]*PrePaidDataUnit)
		if len(params) == 0 {
			return nil, fmt.Errorf("You need to pass the Price ")
		}
		businessObject, err := convertRawDataToPrePaidUsage(data)
		if err != nil {
			return nil, err
		}
		oldState := businessObject.(*PrePaidUsageData)

		newState := oldState.Clone()
		updateDetails(newState, USED_AMOUNT, params[0].(*PrePaidDataUnit))
		if newState.UsedAmount.Cmp(oldState.PlannedAmount.Add(oldState.PlannedAmount, oldState.FailedAmount)) > 0 {
			return nil, fmt.Errorf("Usage Exceeded on channel %v", oldState.ChannelID)
		}
		return BuildOldAndNewValuesForCAS(newState)

	}
	//Make sure you update the planned amount ONLY when the new value is greater than what was last persisted
	IncrementPlannedAmount ConditionFunc = func(params ...interface{}) (newValues []*TypedKeyValueData, err error) {
		data := params[0].([]*PrePaidDataUnit)
		if len(params) == 0 {
			return nil, fmt.Errorf("You need to pass the Price and the Channel Id ")
		}
		businessObject, err := convertRawDataToPrePaidUsage(data)
		if err != nil {
			return nil, err
		}
		oldState := businessObject.(*PrePaidUsageData)
		newState := oldState.Clone()
		updateDetails(newState, PLANNED_AMOUNT, params[0].(*PrePaidDataUnit))
		if newState.PlannedAmount.Cmp(oldState.PlannedAmount) < 0 {
			return nil, fmt.Errorf("A revised higher planned amount has been signed "+
				"already for %v on channel %v", oldState.PlannedAmount, oldState.ChannelID)
		}
		return BuildOldAndNewValuesForCAS(newState)

	}
	//If there is no refund amount yet, put it , else add latest value in DB with the additional refund to be done
	IncrementRefundAmount ConditionFunc = func(params ...interface{}) (newValues []*TypedKeyValueData, err error) {
		data := params[0].([]*PrePaidDataUnit)
		if len(params) == 0 {
			return nil, fmt.Errorf("You need to pass the Price ")
		}
		businessObject, err := convertRawDataToPrePaidUsage(data)
		if err != nil {
			return nil, err
		}
		newState := businessObject.(*PrePaidUsageData)
		updateDetails(newState, REFUND_AMOUNT, params[0].(*PrePaidDataUnit))
		return BuildOldAndNewValuesForCAS(newState)

	}
)

func updateDetails(usageData *PrePaidUsageData, updateUsageType string, details *PrePaidDataUnit) {
	usageData.ChannelID = details.ChannelID
	usageData.UpdateUsageType = updateUsageType
	switch updateUsageType {
	case USED_AMOUNT:
		usageData.UsedAmount.Add(details.Amount, usageData.UsedAmount)
	case PLANNED_AMOUNT:
		usageData.PlannedAmount.Add(details.Amount, usageData.PlannedAmount)
	case REFUND_AMOUNT:
		usageData.FailedAmount.Add(details.Amount, usageData.FailedAmount)
	}
}
