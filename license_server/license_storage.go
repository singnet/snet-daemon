package license_server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v6/storage"
)

type Key interface {
}

type LicenseUsageData struct {
	ChannelID       *big.Int
	ServiceID       string
	Planned         Usage
	Used            Usage
	Refund          Usage
	UpdateUsageType string
}

func (data *LicenseUsageData) GetUsageForUsageType() (Usage, error) {
	switch data.UpdateUsageType {
	case PLANNED:
		return data.Planned, nil
	case REFUND:
		return data.Refund, nil
	case USED:
		return data.Used, nil
	}
	return nil, fmt.Errorf("Unknown Usage Type %v", data.UpdateUsageType)
}

func (data *LicenseUsageData) Clone() *LicenseUsageData {
	return &LicenseUsageData{
		ChannelID:       data.ChannelID,
		ServiceID:       data.ServiceID,
		Planned:         data.Planned.Clone(),
		Used:            data.Used.Clone(),
		Refund:          data.Refund.Clone(),
		UpdateUsageType: data.UpdateUsageType,
	}
}

type LicenseDetailsKey struct {
	ChannelID *big.Int
	ServiceID string
}

func (key *LicenseDetailsKey) String() string {
	return fmt.Sprintf("{ID:%v/%v}", key.ChannelID, key.ServiceID)
}

func serializeLicenseDetailsKey(key any) (serialized string, err error) {
	myKey := key.(LicenseDetailsKey)
	return myKey.String(), nil
}

type LicenseDetailsData struct {
	License License
	// This is to capture the fixed price at the time of purchasing the license
	// If there is no Dynamic Pricing involved, we will need to fall back on the fixed price that is available
	// Keep this flexible to ensure we also support method level pricing in the future
	FixedPricing ServiceMethodCostDetails
}
type LicenseUsageTrackerKey struct {
	ChannelID *big.Int
	ServiceID string
	UsageType string
}

func serializeLicenseUsageTrackerKey(key any) (serialized string, err error) {
	myKey := key.(LicenseUsageTrackerKey)
	return myKey.String(), nil
}

type LicenseUsageTrackerData struct {
	ChannelID *big.Int
	ServiceID string
	Usage     Usage
}

func (data *LicenseUsageTrackerData) String() string {
	return fmt.Sprintf("{ChannelID:%v,ServiceID:%v,Usage:%v}",
		data.ChannelID, data.ServiceID, data.Usage.String())
}
func (key *LicenseUsageTrackerKey) String() string {
	return fmt.Sprintf("{ID:%v/%v/%v}", key.ChannelID, key.ServiceID, key.UsageType)
}

type Discount interface {
	GetDiscount(price *big.Int) *big.Float
	String() string
}
type Usage interface {
	GetUsageType() string
	GetUsage() *big.Int
	SetUsage(*big.Int)
	SetUsageType(string)
	String() string
	Clone() Usage
}

type License interface {
	GetName() string
	GetType() string
	IsActive() bool
	IsCallEligible() (bool, error)
	String() string
	ValidFrom() time.Time
	ValidTo() time.Time
	IsUserEligible(user string) (bool, error)
	GetAddress() []string
}

const (
	USED    string = "U"
	PLANNED string = "P"
	REFUND  string = "R"
)

const (
	SUBSCRIPTION = "SUBSCRIPTION"
	TIER         = "TIER"
	ADD_ON       = "ADD_ON"
	AMOUNT       = "AMOUNT"
	CALLS        = "CALLS"
)

type UsageInAmount struct {
	UsageType string
	Amount    *big.Int
}

type UsageInCalls struct {
	UsageType string
	Calls     *big.Int
}

func (u *UsageInAmount) GetUsageType() string {
	return u.UsageType
}
func (u *UsageInAmount) GetUsage() *big.Int {
	return u.Amount
}
func (u *UsageInAmount) SetUsage(amount *big.Int) {
	u.Amount = amount
}
func (u *UsageInAmount) SetUsageType(typePassed string) {
	u.UsageType = typePassed
}
func (u *UsageInAmount) String() string {
	return fmt.Sprintf("{UsageType:%v,Usage:%v}", u.UsageType, u.Amount)
}
func (u *UsageInAmount) Clone() Usage {
	return &UsageInCalls{Calls: big.NewInt(u.Amount.Int64()), UsageType: u.GetUsageType()}
}
func (u *UsageInCalls) GetUsageType() string {
	return u.UsageType
}
func (u *UsageInCalls) GetUsage() *big.Int {
	return u.Calls
}
func (u *UsageInCalls) Clone() Usage {
	return &UsageInCalls{Calls: big.NewInt(u.Calls.Int64()), UsageType: u.GetUsageType()}
}
func (u *UsageInCalls) String() string {
	return fmt.Sprintf("{UsageType:%v,Usage:%v}", u.UsageType, u.Calls)
}
func (u *UsageInCalls) SetUsage(calls *big.Int) {
	u.Calls = calls
}
func (u *UsageInCalls) SetUsageType(typePassed string) {
	u.UsageType = typePassed
}

type ValidityPeriod struct {
	StartTimeUTC  time.Time
	EndTimeUTC    time.Time
	UpdateTimeUTC time.Time
}

func (u ValidityPeriod) String() string {
	return fmt.Sprintf("{StartTimeUTC:%v,EndTimeUTC:%v.UpdateTimeUTC:%v}",
		u.StartTimeUTC, u.EndTimeUTC, u.UpdateTimeUTC)
}

type AddOn struct {
	ChannelId *big.Int
	Discount  Discount
	//Expiry of AddOn will be tied to the license Type associated with it
	AssociatedLicense License
	Details           *PricingDetails
}

type Subscription struct {
	ChannelId           *big.Int
	ServiceId           string
	Validity            *ValidityPeriod
	Details             *PricingDetails
	Discount            Discount
	AuthorizedAddresses []string
}
type Tier struct {
	Validity            ValidityPeriod
	Details             []TierPricingDetails
	AuthorizedAddresses []string
}

type ServiceMethodCostDetails struct {
	PlanName    string
	ServiceName string
	MethodName  string
	Price       *big.Int
}

func (s ServiceMethodCostDetails) String() string {
	return fmt.Sprintf("{Validity:%v,Details:%v,Discount:%v}",
		s.PlanName, s.ServiceName, s.MethodName)
}

type DiscountPercentage struct {
	DiscountCode    string
	DiscountPercent *big.Float // check if this needs to be big.float
	DiscountName    string
	ValidityPeriod  *ValidityPeriod
}

func (d DiscountPercentage) String() string {
	return fmt.Sprintf("{DiscountCode:%v,DiscountPercent:%v,DiscountName:%v,ValidityPeriod:%v}",
		d.DiscountCode, d.DiscountPercent, d.DiscountName, d.ValidityPeriod)
}

func (d DiscountPercentage) GetDiscount(price *big.Int) *big.Float {
	f := new(big.Float).SetUint64(price.Uint64())
	return f.Mul(f, d.DiscountPercent)
}

type TierPricingDetails struct {
	UpperLimit         *big.Int
	PricePerCall       *big.Int
	ActualAmountSigned *big.Int
}

type PricingDetails struct {
	CreditsInCogs        *big.Int
	FeeInCogs            *big.Int
	LockedPrice          *big.Int // Fixed Price that was defined at the time of creating a license contract
	PlanName             string
	ValidityInDays       uint8
	ActualAmountSigned   *big.Int
	ServiceMethodDetails *ServiceMethodCostDetails // If this is null, implies it applies to all methods of the Service or just the one defined here
}

func (s PricingDetails) String() string {
	return fmt.Sprintf("{CreditsInCogs:%v,FeeInCogs:%v,PlanName:%v"+
		",ValidityInDays:%v,ActualAmountSigned:%v,ServiceMethodCostDetails:%v}",
		s.CreditsInCogs, s.FeeInCogs, s.PlanName, s.ValidityInDays, s.ActualAmountSigned,
		s.ServiceMethodDetails)
}

func (s Subscription) GetName() string {
	return s.Details.PlanName
}

func (s Subscription) ValidFrom() time.Time {
	return s.Validity.StartTimeUTC
}

func (s Subscription) ValidTo() time.Time {
	return s.Validity.EndTimeUTC
}

func (s Subscription) GetAddress() []string {
	return s.AuthorizedAddresses
}

func (s Subscription) GetType() string {
	return SUBSCRIPTION
}
func (s Subscription) IsActive() bool {
	return time.Now().UTC().Before(s.Validity.EndTimeUTC) && time.Now().UTC().After(s.Validity.StartTimeUTC)
}

func (s Subscription) IsCallEligible() (bool, error) {
	return time.Now().UTC().Before(s.Validity.EndTimeUTC) && time.Now().UTC().After(s.Validity.StartTimeUTC), nil
}

func (s Subscription) IsUserEligible(user string) (bool, error) {
	for _, allowed := range s.AuthorizedAddresses {
		if strings.Compare(allowed, user) == 0 {
			return true, nil
		}
	}
	return false, fmt.Errorf("user %s is not in the allowed List of addressed", user)
}

func (s Subscription) String() string {
	return fmt.Sprintf("{Validity:%v,Details:%v,Discount:%v}",
		s.Validity.String(), s.Details.String(), s.Discount.String())
}

func (s Tier) GetType() string {
	return TIER
}
func (s Tier) IsActive() bool {
	return time.Now().UTC().Before(s.Validity.EndTimeUTC) && time.Now().UTC().After(s.Validity.StartTimeUTC)
}

func (s Tier) ValidFrom() time.Time {
	return s.Validity.StartTimeUTC
}
func (s Tier) ValidTo() time.Time {
	return s.Validity.EndTimeUTC
}

func (s Tier) IsUserEligible(user string) (bool, error) {
	for _, allowed := range s.AuthorizedAddresses {
		if strings.Compare(allowed, user) == 0 {
			return true, nil
		}
	}
	return false, fmt.Errorf("user %s is not in the allowed List of addressed", user)
}
func (s Tier) GetAddress() []string {
	return s.AuthorizedAddresses
}

func NewLicenseDetailsStorage(atomicStorage storage.AtomicStorage) storage.TypedAtomicStorage {
	return storage.NewTypedAtomicStorageImpl(
		storage.NewPrefixedAtomicStorage(atomicStorage, "LicenseDetails/storage"),
		serializeLicenseDetailsKey,
		reflect.TypeOf(LicenseDetailsKey{}),
		serializeLicenseDetailsData,
		deserializeLicenseDetailsData,
		reflect.TypeOf(LicenseDetailsData{}))
}

func serializeLicenseDetailsData(value any) (slice string, err error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	gob.Register(&Subscription{})
	gob.Register(&ValidityPeriod{})
	gob.Register(&PricingDetails{})
	gob.Register(&TierPricingDetails{})
	gob.Register(&DiscountPercentage{})
	gob.Register(&ServiceMethodCostDetails{})

	err = e.Encode(value)
	if err != nil {
		return
	}
	return b.String(), err
}

func deserializeLicenseDetailsData(slice string, value any) (err error) {
	b := bytes.NewBuffer([]byte(slice))
	gob.Register(&Subscription{})
	gob.Register(&ValidityPeriod{})
	gob.Register(&PricingDetails{})
	gob.Register(&TierPricingDetails{})
	gob.Register(&DiscountPercentage{})
	gob.Register(&ServiceMethodCostDetails{})

	d := gob.NewDecoder(b)
	err = d.Decode(value)
	return
}

func serializeLicenseTrackerData(value any) (slice string, err error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	gob.Register(&UsageInCalls{})
	err = e.Encode(value)

	if err != nil {
		return
	}

	return b.String(), err
}

func deserializeLicenseTrackerData(slice string, value any) (err error) {
	b := bytes.NewBuffer([]byte(slice))
	d := gob.NewDecoder(b)
	gob.Register(&UsageInCalls{})
	err = d.Decode(value)
	return
}

func NewLicenseUsageTrackerStorage(atomicStorage storage.AtomicStorage) storage.TypedAtomicStorage {
	return storage.NewTypedAtomicStorageImpl(
		storage.NewPrefixedAtomicStorage(atomicStorage, "LicenseUsageTracker/storage"),
		serializeLicenseUsageTrackerKey,
		reflect.TypeOf(LicenseUsageTrackerKey{}),
		serializeLicenseTrackerData,
		deserializeLicenseTrackerData,
		reflect.TypeOf(LicenseUsageTrackerData{}),
	)
}
