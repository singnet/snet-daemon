package pricing

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)



func TestPricing_GetPrice(t *testing.T) {
	metadata, _ := blockchain.ReadServiceMetaDataFromLocalFile("../service_metadata.json")
	metadata.Pricing.PriceModel ="undefined"

	pricing,err := InitPricingStrategy(metadata)
	assert.Equal(t,err.Error(),"No PricingStrategy strategy defined in Metadata ")
	assert.Nil(t,pricing)

	metadata.Pricing.PriceModel =FIXED_PRICING
	pricing,err = InitPricingStrategy(metadata)
	price,err := pricing.GetPrice(nil)
	assert.Equal(t,price,big.NewInt(10000000))
	assert.Nil(t,err)

}


