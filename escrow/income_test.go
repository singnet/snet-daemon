package escrow

import (
	"fmt"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/pricing"
	"google.golang.org/grpc"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

type incomeValidatorMockType struct {
	err error
}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeStreamData) (err error) {
	return incomeValidator.err
}

type MockPriceType struct {
}

func (priceType MockPriceType) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	return big.NewInt(0), nil
}
func (priceType MockPriceType) GetPriceType() string {
	return pricing.FIXED_PRICING
}

var testJsonDataFixedPrice = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\",   \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {       \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",    \"default\":true,         \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",                \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],    \"group_name\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",   \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

var pricingMetadata, _ = blockchain.InitServiceMetaDataFromJson([]byte(testJsonDataFixedPrice))

func TestIncomeValidate(t *testing.T) {
	one := big.NewInt(1)
	income := big.NewInt(0)
	pricingStrt, err := pricing.InitPricingStrategy(pricingMetadata)
	assert.Nil(t, err)
	pricingStrt.AddPricingTypes(&MockPriceType{})
	incomeValidator := NewIncomeStreamValidator(pricingStrt, nil)
	price := big.NewInt(0)

	income.Sub(price, one)
	err = incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Equal(t, NewPaymentError(Unauthenticated, "income %s does not equal to price %s", income, price), err)

	income.Set(price)
	err = incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Nil(t, err)

	income.Add(price, one)
	err = incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Equal(t, NewPaymentError(Unauthenticated, "income %s does not equal to price %s", income, price), err)
}

type MockPriceErrorType struct {
}

func (priceType MockPriceErrorType) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	return nil, fmt.Errorf("Error in Determining Price")
}

func (priceType MockPriceErrorType) GetPriceType() string {
	return pricing.FIXED_PRICING
}
func TestIncomeValidateForPriceError(t *testing.T) {
	pricingStrt, err := pricing.InitPricingStrategy(pricingMetadata)
	assert.Nil(t, err)
	pricingStrt.AddPricingTypes(&MockPriceErrorType{})
	incomeValidator := NewIncomeStreamValidator(pricingStrt, nil)
	err = incomeValidator.Validate(&IncomeStreamData{Income: big.NewInt(0), GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Equal(t, err.Error(), "Error in Determining Price")
}
