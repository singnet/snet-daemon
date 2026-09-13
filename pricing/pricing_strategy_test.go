package pricing

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type testPriceType struct {
	name  string
	price *big.Int
}

func (priceType testPriceType) GetPrice(*handler.GrpcStreamContext) (*big.Int, error) {
	return priceType.price, nil
}

func (priceType testPriceType) GetPriceType() string {
	return priceType.name
}

func TestPricingStrategyAddPricingTypesInitializesRegistry(t *testing.T) {
	strategy := &PricingStrategy{}
	priceType := testPriceType{name: "test", price: big.NewInt(42)}

	strategy.AddPricingTypes(priceType)

	require.Equal(t, priceType, strategy.pricingTypes["test"])
}

func TestPricingStrategyUsesFixedPriceWhenNoDynamicMethodIsMapped(t *testing.T) {
	metadata, err := blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
	require.NoError(t, err)

	originalDynamicPricing := config.GetBool(config.EnableDynamicPricing)
	config.Vip().Set(config.EnableDynamicPricing, true)
	t.Cleanup(func() { config.Vip().Set(config.EnableDynamicPricing, originalDynamicPricing) })

	strategy, err := InitPricingStrategy(metadata)
	require.NoError(t, err)

	priceType, err := strategy.determinePricingApplicable("/example.Service/not_mapped")
	require.NoError(t, err)
	require.Equal(t, FIXED_PRICING, priceType.GetPriceType())

	price, err := strategy.GetPrice(&handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "/example.Service/not_mapped"}})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(2), price)
}

var testJsonData = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\",   \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {       \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",    \"default\":true,         \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",                \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],    \"group_name\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",   \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

func TestPricing_GetPrice(t *testing.T) {
	metadata, _ := blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))

	pricing, err := InitPricingStrategy(metadata)
	if pricing != nil {
		price, err := pricing.GetPrice(&handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "add"}})
		assert.Equal(t, price, big.NewInt(2))
		assert.Nil(t, err)
	}
	assert.Nil(t, err)
	/*metadata, err = blockchain.InitServiceMetaDataFromJson(testJsonData)
	pricing,err = InitPricingStrategy(metadata)
	assert.Equal(t,err.Error(),"No PricingStrategy strategy defined in Metadata ")
	assert.Nil(t,pricing)*/
}
