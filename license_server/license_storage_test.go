package license_server

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

type ServiceMethodDetails1 struct {
	PlanName    string
	ServiceName string
	MethodName  string
}
type ValidityPeriod1 struct {
	StartTimeUTC  *big.Int
	EndTimeUTC    *big.Int
	UpdateTimeUTC time.Time
}
type SubscriptionPricingDetails1 struct {
	CallsAllowed         *big.Int
	FeeInCogs            *big.Int
	PlanName             string
	ValidityInDays       uint8
	ActualAmountSigned   *big.Int
	ServiceMethodDetails *ServiceMethodDetails1 //If this is null , implies it applies to all methods of the Service or just the one defined here
}

func Test_serializeLicenseDetailsData(t *testing.T) {

	validityPeriod := &ValidityPeriod{
		UpdateTimeUTC: time.Now().UTC(),
		StartTimeUTC:  time.Now().UTC(),
		EndTimeUTC:    time.Now().Add(time.Hour * 24).UTC(),
	}

	license := &Subscription{
		ChannelId: big.NewInt(10),
		ServiceId: "sss",
		Validity:  validityPeriod,
		Discount: &DiscountPercentage{
			ValidityPeriod: validityPeriod,
		},

		Details: &PricingDetails{
			PlanName:             "MyTestPlan",
			ActualAmountSigned:   big.NewInt(340),
			ValidityInDays:       120,
			FeeInCogs:            big.NewInt(120),
			CreditsInCogs:        big.NewInt(130),
			ServiceMethodDetails: &ServiceMethodCostDetails{MethodName: "M1", ServiceName: "S1"},
		},
	}
	str, err := serializeLicenseDetailsData(license)
	assert.Nil(t, err)
	subs := &Subscription{}
	err = deserializeLicenseDetailsData(str, subs)
	assert.NotNil(t, str)
	assert.NotNil(t, subs.Validity)
}
