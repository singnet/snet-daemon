package license_server

import (
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func memoryLicenseService() *LockingLicenseService {
	mem := storage.NewMemStorage()
	return NewLicenseService(NewLicenseDetailsStorage(mem), NewLicenseUsageTrackerStorage(mem), nil, nil)
}

func licenseTestMetadata(t *testing.T) *blockchain.ServiceMetadata {
	t.Helper()
	// License tests only need pricing; omit API sources to avoid downloading from IPFS.
	metadata, err := blockchain.InitServiceMetaDataFromJson([]byte(`{
		"service_type": "grpc",
		"mpe_address": "0x34E2EeE197EfAAbEcC495FdF3B1781a3b894eB5f",
		"groups": [{
			"group_name": "default_group",
			"pricing": [{"price_model": "fixed_price", "price_in_cogs": 7, "default": true}]
		}]
	}`))
	require.NoError(t, err)
	return metadata
}

func TestLicenseDetailsLifecycle(t *testing.T) {
	svc := memoryLicenseService()
	metadata := licenseTestMetadata(t)
	svc.ServiceMetaData = metadata
	channel := big.NewInt(9)
	key := LicenseDetailsKey{ChannelID: channel, ServiceID: "service"}
	missing, ok, err := svc.GetLicenseForChannel(key)
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, missing)
	license := &Subscription{ChannelId: channel, ServiceId: "service", Details: &PricingDetails{PlanName: "first"}}
	require.NoError(t, svc.CreateLicenseDetails(channel, "service", license))
	data, ok, err := svc.GetLicenseForChannel(key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, license, data.License)
	require.Equal(t, metadata.GetDefaultPricing().PriceInCogs, data.FixedPricing.Price)
	require.Equal(t, metadata.GetDefaultPricing().PriceModel, data.FixedPricing.PlanName)
	updated := &Subscription{ChannelId: channel, ServiceId: "service", Details: &PricingDetails{PlanName: "second"}}
	require.NoError(t, svc.UpdateLicenseForChannel(channel, "service", updated))
	data, ok, err = svc.GetLicenseForChannel(key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, updated, data.License)
	_, ok, err = svc.GetLicenseForChannel(LicenseDetailsKey{ChannelID: channel, ServiceID: "different"})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestLicenseUsageLimitAndRefund(t *testing.T) {
	svc := memoryLicenseService()
	channel := big.NewInt(5)
	require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(100), PLANNED, SUBSCRIPTION))
	require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(40), USED, SUBSCRIPTION))
	_, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: channel, ServiceID: "service", UsageType: REFUND})
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(4), REFUND, SUBSCRIPTION))
	require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(6), REFUND, SUBSCRIPTION))
	require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(70), USED, SUBSCRIPTION))
	require.ErrorContains(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(1), USED, SUBSCRIPTION), "usage exceeded")
	for kind, want := range map[string]int64{PLANNED: 100, USED: 110, REFUND: 10} {
		data, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: channel, ServiceID: "service", UsageType: kind})
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, big.NewInt(want), data.Usage.GetUsage())
	}
	for _, kind := range []string{USED, REFUND} {
		require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(0), kind, SUBSCRIPTION))
		data, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: channel, ServiceID: "service", UsageType: kind})
		require.NoError(t, err)
		require.True(t, ok)
		require.Zero(t, data.Usage.GetUsage().Sign())
	}
	missing, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: big.NewInt(6), ServiceID: "service", UsageType: USED})
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, missing)
}

func TestConcurrentLicenseUsageDoesNotExceedLimit(t *testing.T) {
	svc := memoryLicenseService()
	channel := big.NewInt(5)
	require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(10), PLANNED, SUBSCRIPTION))
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- svc.UpdateLicenseUsage(channel, "service", big.NewInt(1), USED, SUBSCRIPTION)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else {
			require.ErrorContains(t, err, "usage exceeded")
		}
	}
	require.Equal(t, 10, succeeded)
	data, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: channel, ServiceID: "service", UsageType: USED})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, big.NewInt(10), data.Usage.GetUsage())
}

type licenseStorageFailure struct {
	storage.TypedAtomicStorage
	err error
}

func (s licenseStorageFailure) Get(any) (any, bool, error) { return nil, false, s.err }

func (s licenseStorageFailure) Put(any, any) error { return s.err }

func (s licenseStorageFailure) ExecuteTransaction(storage.TypedCASRequest) (bool, error) {
	return false, s.err
}

func TestLicenseStorageFailures(t *testing.T) {
	sentinel := errors.New("storage unavailable")
	broken := licenseStorageFailure{err: sentinel}
	svc := NewLicenseService(broken, broken, nil, nil)
	data, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{})
	require.ErrorIs(t, err, sentinel)
	require.False(t, ok)
	require.Nil(t, data)
	details, ok, err := svc.GetLicenseForChannel(LicenseDetailsKey{})
	require.ErrorIs(t, err, sentinel)
	require.False(t, ok)
	require.Nil(t, details)
	require.ErrorIs(t, svc.UpdateLicenseForChannel(big.NewInt(1), "service", &Subscription{}), sentinel)
	require.ErrorIs(t, svc.UpdateLicenseUsage(big.NewInt(1), "service", big.NewInt(1), USED, SUBSCRIPTION), sentinel)
	require.ErrorIs(t, svc.CreateOrUpdateLicense(big.NewInt(1), "service"), sentinel)
	require.ErrorContains(t, svc.UpdateLicenseUsage(big.NewInt(1), "service", big.NewInt(1), "invalid", SUBSCRIPTION), "unknown update type")
	svc.LicenseUsageStorage = licenseStorageFailure{}
	require.ErrorContains(t, svc.UpdateLicenseUsage(big.NewInt(1), "service", big.NewInt(1), USED, SUBSCRIPTION), "Error in executing ExecuteTransaction")
}

func TestInvalidUsageTypeIsRejected(t *testing.T) {
	invalid := []storage.TypedKeyValueData{{
		Key:     LicenseUsageTrackerKey{ChannelID: big.NewInt(1), ServiceID: "service", UsageType: "invalid"},
		Present: true, Value: &LicenseUsageTrackerData{Usage: &UsageInCalls{Calls: big.NewInt(1)}},
	}}
	for name, condition := range map[string]ConditionFuncForLicense{USED: IncrementUsedUsage, PLANNED: UpdatePlannedUsage, REFUND: IncrementRefundUsage} {
		t.Run(name, func(t *testing.T) {
			values, err := condition(invalid, big.NewInt(1), big.NewInt(1), "service")
			require.ErrorContains(t, err, "unknown usage type")
			require.Nil(t, values)
		})
	}
	values, err := BuildOldAndNewLicenseUsageValuesForCAS(&LicenseUsageData{UpdateUsageType: "invalid"})
	require.ErrorContains(t, err, "Unknown Usage Type")
	require.Nil(t, values)
}

var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"

type LicenseServiceTestSuite struct {
	suite.Suite
	service               LicenseService
	licenseDetailsStorage storage.TypedAtomicStorage
	licenseUsageStorage   storage.TypedAtomicStorage
	orgMetaData           *blockchain.OrganizationMetaData
	servMetaData          *blockchain.ServiceMetadata
	channelID             *big.Int
}

func (suite *LicenseServiceTestSuite) SetupSuite() {
	suite.channelID = big.NewInt(1)
	suite.orgMetaData, _ = blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	suite.servMetaData = licenseTestMetadata(suite.T())
	suite.licenseDetailsStorage = NewLicenseDetailsStorage(storage.NewMemStorage())
	suite.licenseUsageStorage = NewLicenseUsageTrackerStorage(storage.NewMemStorage())
	suite.service = NewLicenseService(suite.licenseDetailsStorage, suite.licenseUsageStorage, suite.orgMetaData,
		suite.servMetaData)
}

func TestTokenServiceTestSuite(t *testing.T) {
	suite.Run(t, new(LicenseServiceTestSuite))
}

func (suite *LicenseServiceTestSuite) TestCreateLicense() {
	err := suite.service.UpdateLicenseUsage(suite.channelID,
		"serviceId1", big.NewInt(100), PLANNED, "Subscription")
	assert.Nil(suite.T(), err)
	usage, ok, err := suite.service.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: suite.channelID, ServiceID: "serviceId1", UsageType: PLANNED})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), usage.Usage.GetUsage(), big.NewInt(100))
}

func TestLicenseUsageLargePersistedCounters(t *testing.T) {
	for _, kind := range []string{AMOUNT, CALLS} {
		t.Run(kind, func(t *testing.T) {
			svc := memoryLicenseService()
			channel := big.NewInt(9)
			limit := new(big.Int).Lsh(big.NewInt(1), 100)
			for usageType, value := range map[string]*big.Int{
				PLANNED: limit,
				USED:    new(big.Int).Sub(limit, big.NewInt(1)),
				REFUND:  big.NewInt(0),
			} {
				var usage Usage = &UsageInAmount{Amount: value, UsageType: usageType}
				if kind == CALLS {
					usage = &UsageInCalls{Calls: value, UsageType: usageType}
				}
				require.NoError(t, svc.LicenseUsageStorage.Put(
					LicenseUsageTrackerKey{ChannelID: channel, ServiceID: "service", UsageType: usageType},
					&LicenseUsageTrackerData{ChannelID: channel, ServiceID: "service", Usage: usage},
				))
			}
			require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(1), USED, SUBSCRIPTION))
			require.ErrorContains(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(1), USED, SUBSCRIPTION), "usage exceeded")
			require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(2), REFUND, SUBSCRIPTION))
			require.NoError(t, svc.UpdateLicenseUsage(channel, "service", big.NewInt(2), USED, SUBSCRIPTION))
			for usageType, want := range map[string]*big.Int{
				PLANNED: limit,
				USED:    new(big.Int).Add(limit, big.NewInt(2)),
				REFUND:  big.NewInt(2),
			} {
				data, ok, err := svc.GetLicenseUsage(LicenseUsageTrackerKey{ChannelID: channel, ServiceID: "service", UsageType: usageType})
				require.NoError(t, err)
				require.True(t, ok)
				require.Equal(t, want, data.Usage.GetUsage())
				if kind == AMOUNT {
					require.IsType(t, &UsageInAmount{}, data.Usage)
				} else {
					require.IsType(t, &UsageInCalls{}, data.Usage)
				}
			}
		})
	}
}
