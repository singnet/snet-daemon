package price

import (
	"github.com/singnet/snet-daemon/blockchain"
	"google.golang.org/grpc"

	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/handler"
	"github.com/stretchr/testify/assert"
)



func TestFixedMethodPrice_initPricingData(t *testing.T) {
	metadata,_ := blockchain.ReadServiceMetaDataFromLocalFile("../../service_metadata_method_pricing.json")
	grpcCtx := &handler.GrpcStreamContext{Info:&grpc.StreamServerInfo{FullMethod:"/example_service.Calculator/add"}}
	pricing,_ := InitPricing(metadata)
	price,err := pricing.pricingTypes[0].GetPrice(grpcCtx)
	assert.Equal(t,price,big.NewInt(2))
	assert.Nil(t,err)
	grpcCtx.Info.FullMethod= "NonDefinedMethod"
	price,err = pricing.pricingTypes[0].GetPrice(grpcCtx)
	assert.Nil(t,price)
	assert.Equal(t,err.Error(),"price is not defined for the Method NonDefinedMethod")

}


