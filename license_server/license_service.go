package license_server

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
)

type LockingLicenseService struct {
	LicenseDetailsStorage storage.TypedAtomicStorage
	LicenseUsageStorage   storage.TypedAtomicStorage
	Org                   *blockchain.OrganizationMetaData
	ServiceMetaData       *blockchain.ServiceMetadata
}

// NewLicenseService create a new instance of LicenseService
func NewLicenseService(
	detailsStorage storage.TypedAtomicStorage,
	licenseStorage storage.TypedAtomicStorage, orgData *blockchain.OrganizationMetaData,
	serviceMetaData *blockchain.ServiceMetadata) *LockingLicenseService {
	return &LockingLicenseService{
		LicenseDetailsStorage: detailsStorage,
		LicenseUsageStorage:   licenseStorage,
		Org:                   orgData,
		ServiceMetaData:       serviceMetaData,
	}
}

func (h *LockingLicenseService) CreateLicenseDetails(channelId *big.Int, serviceId string,
	license License) (err error) {
	key := LicenseDetailsKey{ServiceID: serviceId, ChannelID: channelId}
	//Get the associated fixed Pricing details
	fixedPricingDetails, err := h.getFixedPriceFromServiceMetadata(license)
	if err != nil {
		return err
	}
	data := LicenseDetailsData{License: license, FixedPricing: fixedPricingDetails}
	return h.LicenseDetailsStorage.Put(key, data)
}

func (h *LockingLicenseService) getFixedPriceFromServiceMetadata(license License) (fixedPricingDetails ServiceMethodCostDetails, err error) {
	fixedPricingDetails = ServiceMethodCostDetails{}
	serviceMetadata := h.ServiceMetaData
	fixedPricingDetails.Price = serviceMetadata.GetDefaultPricing().PriceInCogs
	fixedPricingDetails.PlanName = serviceMetadata.GetDefaultPricing().PriceModel
	return
}

func (h *LockingLicenseService) CreateOrUpdateLicense(channelId *big.Int, serviceId string) (err error) {
	return h.UpdateLicenseUsage(channelId, serviceId, nil, PLANNED, SUBSCRIPTION)
}

func (h *LockingLicenseService) GetLicenseUsage(key LicenseUsageTrackerKey) (*LicenseUsageTrackerData, bool, error) {
	value, ok, err := h.LicenseUsageStorage.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*LicenseUsageTrackerData), ok, err

}

func (h *LockingLicenseService) GetLicenseForChannel(key LicenseDetailsKey) (*LicenseDetailsData, bool, error) {
	value, ok, err := h.LicenseDetailsStorage.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*LicenseDetailsData), ok, err
}

func (h *LockingLicenseService) UpdateLicenseForChannel(channelId *big.Int, serviceId string, license License) error {
	return h.LicenseDetailsStorage.Put(LicenseDetailsKey{ServiceID: serviceId, ChannelID: channelId}, &LicenseDetailsData{License: license})
}

// ConditionFuncForLicense defines the condition that needs to be met, it generates the respective typed Data when
// conditions are satisfied. You define your own validations in here. It takes in the latest typed values read.
type ConditionFuncForLicense func(conditionValues []storage.TypedKeyValueData,
	incrementUsage *big.Int, channelId *big.Int, serviceId string) ([]storage.TypedKeyValueData, error)

func (h *LockingLicenseService) UpdateLicenseUsage(channelId *big.Int, serviceId string, incrementUsage *big.Int, updateUsageType string, licenseType string) error {
	var conditionFunc ConditionFuncForLicense = nil

	switch updateUsageType {
	case USED:
		conditionFunc = IncrementUsedUsage

	case PLANNED:
		conditionFunc = UpdatePlannedUsage

	case REFUND:
		conditionFunc = IncrementRefundUsage

	default:
		return fmt.Errorf("unknown update type %v", updateUsageType)
	}

	typedUpdateFunc := func(conditionValues []storage.TypedKeyValueData) (update []storage.TypedKeyValueData, ok bool, err error) {
		var newValues []storage.TypedKeyValueData
		if newValues, err = conditionFunc(conditionValues, incrementUsage, channelId, serviceId); err != nil {
			return nil, false, err
		}
		return newValues, true, nil
	}
	typedKeys := getAllLicenseKeys(channelId, serviceId)
	request := storage.TypedCASRequest{
		Update:                  typedUpdateFunc,
		RetryTillSuccessOrError: true,
		ConditionKeys:           typedKeys,
	}
	ok, err := h.LicenseUsageStorage.ExecuteTransaction(request)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Error in executing ExecuteTransaction for usage type"+
			"  %v on channel %v ", updateUsageType, channelId)
	}
	return nil
}
func getAllLicenseKeys(channelId *big.Int, serviceId string) []any {
	keys := make([]interface{}, 3)
	for i, usageType := range []string{REFUND, PLANNED, USED} {
		keys[i] = LicenseUsageTrackerKey{ChannelID: channelId, ServiceID: serviceId, UsageType: usageType}
	}
	return keys
}

// this function will be used to read typed data ,convert it in to a business structure
// on which validations can be easily performed and return back the business structure.
func convertTypedDataToLicenseDataUsage(data []storage.TypedKeyValueData) (new *LicenseUsageData, err error) {
	usageData := &LicenseUsageData{
		Planned: &UsageInAmount{Amount: big.NewInt(0), UsageType: PLANNED},
		Used:    &UsageInAmount{Amount: big.NewInt(0), UsageType: USED},
		Refund:  &UsageInAmount{Amount: big.NewInt(0), UsageType: REFUND},
	}
	for _, usageType := range data {
		key := usageType.Key.(LicenseUsageTrackerKey)
		usageData.ChannelID = key.ChannelID
		usageData.ServiceID = key.ServiceID
		if !usageType.Present {
			continue
		}
		data := usageType.Value.(*LicenseUsageTrackerData)
		if strings.Compare(key.UsageType, USED) == 0 {
			usageData.Used = data.Usage
		} else if strings.Compare(key.UsageType, PLANNED) == 0 {
			usageData.Planned = data.Usage
		} else if strings.Compare(key.UsageType, REFUND) == 0 {
			usageData.Refund = data.Usage
		} else {
			return nil, fmt.Errorf("unknown usage type %v", key.UsageType)
		}
	}
	return usageData, nil
}

func BuildOldAndNewLicenseUsageValuesForCAS(data *LicenseUsageData) (newValues []storage.TypedKeyValueData, err error) {
	updateUsageData := &LicenseUsageTrackerData{ChannelID: data.ChannelID, ServiceID: data.ServiceID}
	updateUsageKey := LicenseUsageTrackerKey{ChannelID: data.ChannelID, ServiceID: data.ServiceID,
		UsageType: data.UpdateUsageType}
	if usage, err := data.GetUsageForUsageType(); err != nil {
		return nil, err
	} else {
		updateUsageData.Usage = usage
	}
	newValue := storage.TypedKeyValueData{Key: updateUsageKey, Value: updateUsageData, Present: true}
	newValues = make([]storage.TypedKeyValueData, 1)
	newValues[0] = newValue

	return newValues, nil
}

var (
	IncrementUsedUsage ConditionFuncForLicense = func(conditionValues []storage.TypedKeyValueData, incrementUsage *big.Int,
		channelId *big.Int, serviceId string) (newValues []storage.TypedKeyValueData, err error) {
		oldState, err := convertTypedDataToLicenseDataUsage(conditionValues)
		if err != nil {
			return nil, err
		}
		oldState.ChannelID = channelId
		oldState.ServiceID = serviceId
		newState := oldState.Clone()
		usageKey := LicenseUsageTrackerKey{UsageType: USED, ChannelID: oldState.ChannelID, ServiceID: serviceId}
		if incrementUsage.Cmp(big.NewInt(0)) > 0 {
			updateLicenseUsageData(newState, usageKey, incrementUsage)
			if newState.Used.GetUsage().Cmp(oldState.Planned.GetUsage().Add(oldState.Planned.GetUsage(), oldState.Refund.GetUsage())) > 0 {
				return nil, fmt.Errorf("usage exceeded on channel Id %v", oldState.ChannelID)
			}
		} else {
			newState.UpdateUsageType = USED
			newState.Used.SetUsage(big.NewInt(0))
		}
		return BuildOldAndNewLicenseUsageValuesForCAS(newState)

	}
	//Make sure you update the planned amount ONLY when the new value is greater than what was last persisted
	UpdatePlannedUsage ConditionFuncForLicense = func(conditionValues []storage.TypedKeyValueData, incrementUsage *big.Int,
		channelId *big.Int, serviceId string) (newValues []storage.TypedKeyValueData, err error) {
		oldState, err := convertTypedDataToLicenseDataUsage(conditionValues)
		if err != nil {
			return nil, err
		}
		//Assuming there are no entries yet on this channel, it is very easy to pass the channel ID to the condition
		//function and pick it from there
		oldState.ChannelID = channelId
		oldState.ServiceID = serviceId
		newState := oldState.Clone()
		usageKey := LicenseUsageTrackerKey{UsageType: PLANNED, ChannelID: oldState.ChannelID, ServiceID: serviceId}
		updateLicenseUsageData(newState, usageKey, incrementUsage)
		return BuildOldAndNewLicenseUsageValuesForCAS(newState)

	}
	// IncrementRefundUsage If there is no refund amount yet, put it, else add the latest value in DB with the additional refund to be done
	IncrementRefundUsage ConditionFuncForLicense = func(conditionValues []storage.TypedKeyValueData, incrementUsage *big.Int,
		channelId *big.Int, serviceId string) (newValues []storage.TypedKeyValueData, err error) {
		newState, err := convertTypedDataToLicenseDataUsage(conditionValues)
		if err != nil {
			return nil, err
		}
		newState.ChannelID = channelId
		newState.ServiceID = serviceId
		usageKey := LicenseUsageTrackerKey{UsageType: REFUND, ChannelID: channelId, ServiceID: serviceId}
		if incrementUsage.Cmp(big.NewInt(0)) > 0 {
			updateLicenseUsageData(newState, usageKey, incrementUsage)
		} else {
			newState.UpdateUsageType = REFUND
			newState.Refund.SetUsage(big.NewInt(0))
		}
		return BuildOldAndNewLicenseUsageValuesForCAS(newState)

	}
)

func updateLicenseUsageData(usageData *LicenseUsageData, key LicenseUsageTrackerKey, usage *big.Int) {
	usageData.ChannelID = key.ChannelID
	usageData.UpdateUsageType = key.UsageType
	var oldUsage *big.Int
	switch key.UsageType {
	case USED:
		{
			oldUsage = usageData.Used.GetUsage()
			usageData.Used.SetUsage(oldUsage.Add(oldUsage, usage))
		}
	case PLANNED:
		//to reset the counter, Planned Amount will be updated ONLY when the License is Purchased or Renewed
		{
			usageData.Planned.SetUsage(usage)
			usageData.Used.SetUsage(big.NewInt(0))
			usageData.Refund.SetUsage(big.NewInt(0))
		}
	case REFUND:
		{
			oldUsage = usageData.Refund.GetUsage()
			usageData.Refund.SetUsage(oldUsage.Add(oldUsage, usage))
		}
	}
}
