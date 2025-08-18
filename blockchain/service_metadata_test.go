package blockchain

import (
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

var testLicenseJsonData = "\n  \"licenses\": { \"tiers\": [{\n  \"type\": \"Tier\",\n  \"planName\": \"Tier AAA\",\n  \"grpcServiceName\": \"ServiceA\",\n  \"grpcMethodName\": \"MethodA\",\n  \"range\": [\n    {\n      \"high\": 100,\n      \"discountInPercentage\": 1\n    },\n    {\n      \"high\": 200,\n      \"discountInPercentage\": 20\n    },\n    {\n      \"high\": 300,\n      \"discountInPercentage\": 100000\n    }\n  ],\n  \"detailsUrl\": \"http://abc.org/licenses/Tier.html\",\n  \"isActive\": \"true/false\"\n},\n {\n  \"type\": \"Tier\",\n  \"planName\": \"Tier BBB Applicable for All service.methods\",\n  \"range\": [\n    {\n      \"high\": 100,\n      \"discountInPercentage\": 1\n    },\n    {\n      \"high\": 200,\n      \"discountInPercentage\": 200\n    },\n    {\n      \"high\": 300,\n      \"discountInPercentage\": 100000\n    }\n  ],\n  \"detailsUrl\": \"http://abc.org/licenses/Tier.html\",\n  \"isActive\": \"true/false\"\n}], " +
	"\"subscriptions\": {\n   \"subscription\": [\n  {\n    \"periodInDays\": 30,\n    \"discountInPercentage\": 120,\n    \"planName\": \"Monthly For ServiceA/MethodA\",\n    \"LicenseCost\": 90,\n    \"grpcServiceName\": \"ServiceA\",\n    \"grpcMethodName\": \"MethodA\"\n  },\n  {\n    \"periodInDays\": 30,\n    \"discountInPercentage\": 123,\n    \"planName\": \"Monthly\",\n    \"LicenseCost\": 93\n  },\n  {\n    \"periodInDays\": 120,\n    \"discountInPercentage\": 160,\n    \"LicenseCost\": 120,\n    \"planName\": \"Quarterly\"\n  },\n  {\n    \"periodInDays\": 365,\n    \"discountInPercentage\": 430,\n    \"LicenseCost\": 390,\n    \"planName\": \"Yearly\"\n  }\n],       \"type\": \"Subscription\",\n          \"detailsUrl\": \"http://abc.org/licenses/Subscription.html\",\n          \"isActive\": \"true/false\"\n        }\n      }"
var testJsonData = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
	"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",  " + testLicenseJsonData + " ,  \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

func TestAllGetterMethods(t *testing.T) {
	metaData, err := InitServiceMetaDataFromJson([]byte(testJsonData))
	assert.Equal(t, err, nil)

	assert.Equal(t, metaData.GetVersion(), 1)
	assert.Equal(t, metaData.GetDisplayName(), "Example1")
	assert.Equal(t, metaData.GetServiceType(), "grpc")
	assert.Equal(t, metaData.GetWireEncoding(), "grpc")
	assert.Nil(t, metaData.GetDefaultPricing().PriceInCogs)
	assert.Equal(t, metaData.GetDefaultPricing().PricingDetails[0].MethodPricing[0].PriceInCogs, big.NewInt(2))
	assert.Equal(t, metaData.GetMpeAddress(), common.HexToAddress("0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08"))
	assert.Equal(t, metaData.FreeCallSignerAddress(), common.HexToAddress("0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F"))
	assert.True(t, metaData.IsFreeCallAllowed())
	assert.Equal(t, 12, metaData.GetFreeCallsAllowed())
	assert.Equal(t, metaData.GetLicenses().Subscriptions.Type, "Subscription")
}

func TestSubscription(t *testing.T) {
	metaData, err := InitServiceMetaDataFromJson([]byte(testJsonData))
	assert.Equal(t, err, nil)
	assert.Equal(t, 12, metaData.GetFreeCallsAllowed())
	assert.Equal(t, metaData.GetLicenses().Subscriptions.Type, "Subscription")
	assert.Equal(t, len(metaData.GetLicenses().Subscriptions.Subscription), 4)
	assert.Equal(t, metaData.GetLicenses().Subscriptions.Subscription[0].PlanName, "Monthly For ServiceA/MethodA")
	assert.Equal(t, metaData.GetLicenses().Subscriptions.Subscription[0].GrpcMethodName, "MethodA")
	assert.Equal(t, metaData.GetLicenses().Subscriptions.Subscription[0].GrpcServiceName, "ServiceA")
	assert.Equal(t, metaData.GetLicenses().Subscriptions.Subscription[0].DiscountInPercentage, 120.00)
}

func TestTiers(t *testing.T) {
	metaData, err := InitServiceMetaDataFromJson([]byte(testJsonData))
	assert.Equal(t, err, nil)

	assert.Equal(t, metaData.GetLicenses().Tiers[0].Type, "Tier")
	assert.Equal(t, metaData.GetLicenses().Tiers[0].Range[0].High, 100)
	assert.Equal(t, metaData.GetLicenses().Tiers[0].Range[0].DiscountInPercentage,
		1.0)
}

func TestInitServiceMetaDataFromJson(t *testing.T) {
	//Parse Bad JSON
	_, err := InitServiceMetaDataFromJson([]byte(strings.Replace(testJsonData, "{", "", 1)))
	if err != nil {
		assert.Equal(t, err.Error(), "invalid character ':' after top-level value")
	}

	//Parse Bad JSON
	_, err = InitServiceMetaDataFromJson([]byte(strings.Replace(testJsonData, "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F", "", 1)))
	if err != nil {
		assert.Contains(t, err.Error(), "metadata does not have 'free_call_signer_address defined correctly")
	}
	_, err = InitServiceMetaDataFromJson([]byte(strings.Replace(testJsonData, "default_pricing", "dummy", 1)))
	if err != nil {
		assert.Equal(t, err.Error(), "metadata does not have the default pricing set")
	}

}

func TestReadServiceMetaDataFromLocalFile(t *testing.T) {
	metadata, err := ReadServiceMetaDataFromLocalFile("../resources/testing/service_metadata.json")
	assert.Equal(t, err, nil)
	assert.Equal(t, metadata.Version, 1)
}

func Test_getServiceMetaDataUrifromRegistry(t *testing.T) {
	config.Validate()
	_, err := getServiceMetaDataURIFromRegistry()
	assert.NotNil(t, err)
	config.Vip().Set(config.ServiceId, "sepolia")
	config.Vip().Set(config.ServiceId, "semyon_dev")
	config.Vip().Set(config.OrganizationId, "semyon_dev")
	_, err = getServiceMetaDataURIFromRegistry()
	assert.Nil(t, err)
}

func Test_setDefaultPricing(t *testing.T) {
	// Test with empty metadata
	meta := &ServiceMetadata{}
	err := meta.setDefaultPricing()
	assert.NotNil(t, err)
	assert.Equal(t, "group name default_group in config is invalid, there was no group found with this name in the metadata", err.Error())

	// Test with a group, but default pricing is not set
	meta = &ServiceMetadata{
		Groups: []OrganizationGroup{{GroupName: "default_group"}},
	}
	err = meta.setDefaultPricing()
	assert.NotNil(t, err)
	assert.Equal(t, "metadata does not have the default pricing set", err.Error())

	// Test with a group, but pricing is set
	meta = &ServiceMetadata{
		Groups: []OrganizationGroup{{GroupName: "default_group", Pricing: []Pricing{{PriceModel: "fixed_price", Default: true, PriceInCogs: big.NewInt(2)}}}},
	}
	err = meta.setDefaultPricing()
	assert.Nil(t, err)
}

func Test_setGroup(t *testing.T) {
	// Test with empty metadata should return an error
	meta := &ServiceMetadata{}
	err := meta.setGroup()
	assert.NotNil(t, err)
	assert.Equal(t, "group name default_group in config is invalid, there was no group found with this name in the metadata", err.Error())
}

func TestServiceMetadata_parseServiceProto(t *testing.T) {
	strProto := "syntax = \"proto3\";\n\nimport \"singnet/snet-daemon/training/training.proto\";\nimport \"singnet/snet-daemon/pricing/pricing.proto\";\npackage example_service;\n\nmessage AddressList {\n    repeated string address = 1;\n    string model_details = 2;\n}\n\nmessage Numbers {\n    float a = 1;\n    float b = 2;\n    //highlight this as part of documentation\n    string model_id = 3;\n}\n\nmessage Result {\n    float value = 1;\n}\nmessage ModelId {\n    string model_id = 1;\n}\nmessage TrainingRequest  {\n    string link = 1;\n    string address = 2;\n    string model_id =3 ;\n    string request_id =4;\n}\nmessage TrainingResponse  {\n    string link = 1;\n    string model_id =2 ;\n    string status = 3;//put this in enum\n    string request_id = 4;\n}\n\nservice Calculator {\n    //AI developer can define their own training methods , if you wish to have a different pricing , please use\n    // define the pricing method as shown below, dynamic_pricing_train_add, please make sure the i/p is exactly the same\n    rpc train_add(TrainingRequest) returns (TrainingResponse) {\n        option (pricing.my_method_option).estimatePriceMethod = \"/example_service.Calculator/dynamic_pricing_train_add\";\n        option (training.my_method_option).trainingMethodIndicator = \"true\";\n    }\n\n    rpc train_subtraction(TrainingRequest) returns (TrainingResponse) {\n        option (pricing.my_method_option).estimatePriceMethod = \"/example_service.Calculator/dynamic_pricing_train_add\";\n        option (training.my_method_option).trainingMethodIndicator = \"true\";\n    }\n\n    //make sure the params to the method are exactly the same as that of the method in which needs dynamic price computation\n    rpc dynamic_pricing_train_add(TrainingRequest) returns (pricing.PriceInCogs) {}\n    rpc dynamic_pricing_add(Numbers) returns (pricing.PriceInCogs) {}\n\n    rpc add(Numbers) returns (Result) {option (pricing.my_method_option).estimatePriceMethod = \"/example_service.Calculator/dynamic_pricing_add\";}\n    rpc sub(Numbers) returns (Result) {}\n    rpc mul(Numbers) returns (Result) {}\n    rpc div(Numbers) returns (Result) {}\n}\n"
	srvProto, err := parseServiceProto(strProto)
	assert.Nil(t, err)
	priceMethodMap, trainingMethods, err := buildDynamicPricingMethodsMap(srvProto)
	assert.Nil(t, err)
	assert.NotNil(t, priceMethodMap)
	assert.NotNil(t, trainingMethods)
	dynamicPriceMethod, ok := priceMethodMap["/example_service.Calculator/add"]
	isTrainingMethod := slices.Contains(trainingMethods, "/example_service.Calculator/train_add")
	assert.Equal(t, dynamicPriceMethod, "/example_service.Calculator/dynamic_pricing_add")
	assert.True(t, ok, "true")
	assert.True(t, isTrainingMethod)
}

func TestServiceMetadata_addOns(t *testing.T) {
	metadata, err := ReadServiceMetaDataFromLocalFile("../resources/testing/service_metadata.json")
	assert.Equal(t, err, nil)
	assert.Equal(t, metadata.Groups[0].AddOns[0].DiscountInPercentage, 4.0)
}
