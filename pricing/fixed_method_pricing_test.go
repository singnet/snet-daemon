package pricing

import (
	"github.com/singnet/snet-daemon/v6/blockchain"
	"google.golang.org/grpc"

	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
)

var testJsonDataFixedMethodPrice = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\",   \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {       \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",       \"default\":true,           \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

func TestFixedMethodPrice_initPricingData(t *testing.T) {
	metadata, _ := blockchain.InitServiceMetaDataFromJson([]byte(testJsonDataFixedMethodPrice))
	grpcCtx := &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "/example_service.Calculator/add"}}
	pricing, _ := InitPricingStrategy(metadata)
	price, err := pricing.GetPrice(grpcCtx)
	assert.Equal(t, price, big.NewInt(2))
	assert.Nil(t, err)
	//Test with an undefined method Name
	grpcCtx.Info.FullMethod = "NonDefinedMethod"
	price, err = pricing.GetPrice(grpcCtx)
	assert.Nil(t, price)
	if err != nil {
		assert.Equal(t, err.Error(), "price is not defined for the Method NonDefinedMethod")
	}
	/*	//Test if the metadata is not properly defined
		//metadata.GetDefaultPricing().Details = nil

		pricing,err = InitPricingStrategy(metadata)
		assert.Equal(t,err.Error(),"service / method level pricing is not defined correctly")
		assert.Nil(t,pricing)*/

}
