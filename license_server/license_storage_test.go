package license_server

import (
	"io"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseValidityAndAllowedUsers(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name       string
		start, end time.Time
		active     bool
	}{
		{"active", now.Add(-time.Hour), now.Add(time.Hour), true},
		{"expired", now.Add(-2 * time.Hour), now.Add(-time.Hour), false},
		{"future", now.Add(time.Hour), now.Add(2 * time.Hour), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			period := ValidityPeriod{StartTimeUTC: tc.start, EndTimeUTC: tc.end, UpdateTimeUTC: now}
			subscription := Subscription{Validity: &period, Details: &PricingDetails{PlanName: "subscription"}, AuthorizedAddresses: []string{"alice", "bob"}}
			tier := Tier{Validity: period, AuthorizedAddresses: []string{"alice", "bob"}}
			require.Equal(t, tc.active, subscription.IsActive())
			eligible, err := subscription.IsCallEligible()
			require.NoError(t, err)
			require.Equal(t, tc.active, eligible)
			require.Equal(t, tc.active, tier.IsActive())
			require.Equal(t, tc.start, subscription.ValidFrom())
			require.Equal(t, tc.end, subscription.ValidTo())
			require.Equal(t, tc.start, tier.ValidFrom())
			require.Equal(t, tc.end, tier.ValidTo())
			require.Equal(t, SUBSCRIPTION, subscription.GetType())
			require.Equal(t, TIER, tier.GetType())
			require.Equal(t, "subscription", subscription.GetName())
			require.Equal(t, []string{"alice", "bob"}, subscription.GetAddress())
			require.Equal(t, []string{"alice", "bob"}, tier.GetAddress())
			for _, check := range []func(string) (bool, error){subscription.IsUserEligible, tier.IsUserEligible} {
				eligible, err := check("bob")
				require.NoError(t, err)
				require.True(t, eligible)
				eligible, err = check("unknown")
				require.ErrorContains(t, err, "unknown")
				require.False(t, eligible)
			}
		})
	}
}

func TestUsageAccessorsAndCloneIsolation(t *testing.T) {
	for _, usage := range []Usage{&UsageInAmount{Amount: big.NewInt(42)}, &UsageInCalls{Calls: big.NewInt(42)}} {
		usage.SetUsageType(USED)
		require.Equal(t, USED, usage.GetUsageType())
		require.Equal(t, big.NewInt(42), usage.GetUsage())
		copy := usage.Clone()
		copy.GetUsage().SetInt64(7)
		require.Equal(t, big.NewInt(42), usage.GetUsage(), "cloning must not share a mutable counter")
		copy.SetUsage(big.NewInt(9))
		copy.SetUsageType(REFUND)
		require.Equal(t, USED, usage.GetUsageType())
		require.Equal(t, REFUND, copy.GetUsageType())
		require.Equal(t, "{UsageType:R,Usage:9}", copy.String())
		usage.SetUsage(big.NewInt(43))
		require.Equal(t, "{UsageType:U,Usage:43}", usage.String())
	}
}

func TestLicenseDiagnosticStrings(t *testing.T) {
	period := &ValidityPeriod{StartTimeUTC: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	method := &ServiceMethodCostDetails{PlanName: "plan", ServiceName: "service", MethodName: "method"}
	discount := &DiscountPercentage{DiscountCode: "code", DiscountPercent: big.NewFloat(0.25), DiscountName: "discount", ValidityPeriod: period}
	pricing := &PricingDetails{CreditsInCogs: big.NewInt(100), FeeInCogs: big.NewInt(20), PlanName: "plan", ServiceMethodDetails: method}
	license := &Subscription{Validity: period, Discount: discount, Details: pricing}
	for _, text := range []string{"plan", "discount", "code", "2024-01-01"} {
		require.Contains(t, license.String(), text)
	}
	data := &LicenseUsageTrackerData{ChannelID: big.NewInt(9), ServiceID: "service", Usage: &UsageInCalls{UsageType: USED, Calls: big.NewInt(3)}}
	require.Equal(t, "{ChannelID:9,ServiceID:service,Usage:{UsageType:U,Usage:3}}", data.String())
	require.Zero(t, discount.GetDiscount(big.NewInt(80)).Cmp(big.NewFloat(20)))
}

func TestLicenseSerializationRejectsInvalidData(t *testing.T) {
	for name, serialize := range map[string]func(any) (string, error){"details": serializeLicenseDetailsData, "usage": serializeLicenseTrackerData} {
		t.Run(name, func(t *testing.T) {
			data, err := serialize(make(chan int))
			require.Error(t, err)
			require.Empty(t, data)
		})
	}
	require.Error(t, deserializeLicenseDetailsData("invalid gob", &LicenseDetailsData{}))
	require.Error(t, deserializeLicenseTrackerData("invalid gob", &LicenseUsageTrackerData{}))
	key, err := serializeLicenseDetailsKey(LicenseDetailsKey{ChannelID: big.NewInt(9), ServiceID: "service"})
	require.NoError(t, err)
	require.Equal(t, "{ID:9/service}", key)
}

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

func TestUsageClonePreservesTypeAndValue(t *testing.T) {
	large := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 100), big.NewInt(17))
	for _, value := range []*big.Int{big.NewInt(0), big.NewInt(42), new(big.Int).Lsh(big.NewInt(1), 63), large, new(big.Int).Neg(large)} {
		for _, kind := range []string{AMOUNT, CALLS} {
			t.Run(kind+"/"+value.String(), func(t *testing.T) {
				var original Usage = &UsageInAmount{Amount: new(big.Int).Set(value), UsageType: USED}
				if kind == CALLS {
					original = &UsageInCalls{Calls: new(big.Int).Set(value), UsageType: USED}
				}
				cloned := original.Clone()
				require.Equal(t, value, cloned.GetUsage())
				require.IsType(t, original, cloned)
				require.Equal(t, USED, cloned.GetUsageType())
				cloned.GetUsage().Add(cloned.GetUsage(), big.NewInt(1))
				cloned.SetUsageType(REFUND)
				require.Equal(t, value, original.GetUsage())
				require.Equal(t, USED, original.GetUsageType())
				original.GetUsage().Sub(original.GetUsage(), big.NewInt(1))
				require.Equal(t, new(big.Int).Add(value, big.NewInt(1)), cloned.GetUsage())
			})
		}
	}
}

func TestLicenseUsageGobRoundTrip(t *testing.T) {
	value := new(big.Int).Lsh(big.NewInt(1), 100)
	// Decode in a fresh process: encoding must not mask a missing decoder registration.
	if kind := os.Getenv("SNET_LICENSE_GOB_DECODE"); kind != "" {
		encoded, err := io.ReadAll(os.Stdin)
		require.NoError(t, err)
		var decoded LicenseUsageTrackerData
		require.NoError(t, deserializeLicenseTrackerData(string(encoded), &decoded))
		require.Equal(t, big.NewInt(9), decoded.ChannelID)
		require.Equal(t, "service", decoded.ServiceID)
		require.Equal(t, REFUND, decoded.Usage.GetUsageType())
		require.Equal(t, value, decoded.Usage.GetUsage())
		if kind == AMOUNT {
			require.IsType(t, &UsageInAmount{}, decoded.Usage)
		} else {
			require.IsType(t, &UsageInCalls{}, decoded.Usage)
		}
		return
	}
	for _, kind := range []string{AMOUNT, CALLS} {
		t.Run(kind, func(t *testing.T) {
			var usage Usage = &UsageInAmount{Amount: value, UsageType: REFUND}
			if kind == CALLS {
				usage = &UsageInCalls{Calls: value, UsageType: REFUND}
			}
			encoded, err := serializeLicenseTrackerData(&LicenseUsageTrackerData{
				ChannelID: big.NewInt(9), ServiceID: "service", Usage: usage,
			})
			require.NoError(t, err)
			executable, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.Command(executable, "-test.run=^TestLicenseUsageGobRoundTrip$", "-test.timeout=30s")
			cmd.Env = append(os.Environ(), "SNET_LICENSE_GOB_DECODE="+kind)
			cmd.Stdin = strings.NewReader(encoded)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
		})
	}
}
