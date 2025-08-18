package license_server

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

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
	var err error
	suite.channelID = big.NewInt(1)
	suite.orgMetaData, _ = blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	suite.servMetaData, err = blockchain.ReadServiceMetaDataFromLocalFile("../resources/testing/service_metadata.json")
	assert.Nil(suite.T(), err)
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
