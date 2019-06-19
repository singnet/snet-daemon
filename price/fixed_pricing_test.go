package price

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/handler"
)

func TestFixedPrice_GetPrice(t *testing.T) {
	metadata, _ := blockchain.ReadServiceMetaDataFromLocalFile("../service_metadata.json")
	grpcCtx := &handler.GrpcStreamContext{}
	pricing,_ := InitPricingStrategy(metadata)

	price,err := pricing.pricingTypes[0].GetPrice(grpcCtx)
	assert.Equal(t,price,big.NewInt(10000000))
	assert.Nil(t,err)
}
