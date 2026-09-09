package pricing

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
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

func TestFixedMethodPriceReportsUnknownMethod(t *testing.T) {
	priceType := FixedMethodPrice{methodToPriceMap: map[string]*big.Int{"/example.Service/known": big.NewInt(1)}}

	price, err := priceType.GetPrice(&handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "/example.Service/unknown"}})

	require.Nil(t, price)
	require.EqualError(t, err, "price is not defined for the Method /example.Service/unknown")
}
