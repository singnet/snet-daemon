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

//Defines the condition that needs to be met, it generates the new business struct if all validations
//conditions are satisfied, you define your own validations in here
//It takes in the latest values read on the Key Passed, Please note all keys with this prefix will be read
type ConditionFunc func(latestReadData interface{}, params ...interface{}) (newValues interface{}, err error)

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

	transaction, err := h.storage.StartTransaction(channelId.String())
	if err != nil {
		return err
	}

	for {
		var newValues interface{}
		var newKeyValues []*TypedKeyValueData

		if newValues, err = conditionFunc(transaction.GetConditionValues(), revisedAmount); err != nil {
			return err
		}
		if newKeyValues, err = BuildOldAndNewValuesForCAS(newValues); err != nil {
			return err
		}
		ok, err := h.storage.CompleteTransaction(transaction, newKeyValues)
		if err != nil {
			return err
		}
		if ok {
			break
		}
	}

	return nil
}

var (
	//this function will be used to read data from Storage and convert it in to a business structure
	//on which validations can be easily performed.
	convertRawDataToPrePaidUsage = func(latestReadData interface{}) (new interface{}, err error) {
		data := latestReadData.([]*KeyValueData)
		usageData := &PrePaidUsageData{PlannedAmount: big.NewInt(0),
			UsedAmount: big.NewInt(0), FailedAmount: big.NewInt(0)}
		for _, dataRetrieved := range data {
			if dataRetrieved == nil {
				continue
			}
			usageType := &PrePaidDataUnit{}
			if err = deserialize(dataRetrieved.Value, usageType); err != nil {
				return nil, err
			}
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
	IncrementUsedAmount ConditionFunc = func(latestReadData interface{},
		params ...interface{}) (new interface{}, err error) {
		data := latestReadData.([]*KeyValueData)
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
		setLastUpdatedVersion(data, newState, USED_AMOUNT)
		return newState, nil

	}
	//Make sure you update the planned amount ONLY when the new value is greater than what was last persisted
	IncrementPlannedAmount ConditionFunc = func(latestReadData interface{}, params ...interface{}) (new interface{}, err error) {
		data := latestReadData.([]*KeyValueData)
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
		setLastUpdatedVersion(data, newState, PLANNED_AMOUNT)
		return newState, nil

	}
	//If there is no refund amount yet, put it , else add latest value in DB with the additional refund to be done
	IncrementRefundAmount ConditionFunc = func(latestReadData interface{}, params ...interface{}) (new interface{}, err error) {
		data := latestReadData.([]*KeyValueData)
		if len(params) == 0 {
			return nil, fmt.Errorf("You need to pass the Price ")
		}
		businessObject, err := convertRawDataToPrePaidUsage(data)
		if err != nil {
			return nil, err
		}
		usageData := businessObject.(*PrePaidUsageData)
		updateDetails(usageData, REFUND_AMOUNT, params[0].(*PrePaidDataUnit))
		setLastUpdatedVersion(data, usageData, REFUND_AMOUNT)
		return usageData, nil

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

func setLastUpdatedVersion(data []*KeyValueData, usageData *PrePaidUsageData, updateUsageType string) {
	for _, data := range data {
		if data == nil {
			continue
		}
		if strings.Contains(data.Key, updateUsageType) {
			usageData.UpdateUsageType = updateUsageType
		}
	}
}
