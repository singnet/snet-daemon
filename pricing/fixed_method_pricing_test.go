package pricing

import (
	"github.com/singnet/snet-daemon/blockchain"
	"google.golang.org/grpc"

	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/handler"
	"github.com/stretchr/testify/assert"
)

var testJsonDataFixedMethodPrice = "{\"version\": 1, \"display_name\": \"Example1\", \"encoding\": \"grpc\", \"service_type\": \"grpc\", \"payment_expiration_threshold\": 40320, \"model_ipfs_hash\": \"QmQC9EoVdXRWmg8qm25Hkj4fG79YAgpNJCMDoCnknZ6VeJ\", \"mpe_address\": \"0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E\", \"pricing\":{\"package_name\":\"example_service\",\"price_model\":\"fixed_price_per_method\",\"details\":[{\"service_name\":\"Calculator\",\"method_pricing\":[{\"method_name\":\"add\",\"price_in_cogs\":2},{\"method_name\":\"sub\",\"price_in_cogs\":1},{\"method_name\":\"div\",\"price_in_cogs\":2},{\"method_name\":\"mul\",\"price_in_cogs\":3}]},{\"service_name\":\"Calculator2\",\"method_pricing\":[{\"method_name\":\"add\",\"price_in_cogs\":2},{\"method_name\":\"sub\",\"price_in_cogs\":1},{\"method_name\":\"div\",\"price_in_cogs\":3},{\"method_name\":\"mul\",\"price_in_cogs\":2}]}]}, \"groups\": [{\"group_name\": \"default_group\", \"group_id\": \"nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=\", \"payment_address\": \"0xD6C6344f1D122dC6f4C1782A4622B683b9008081\"}], \"endpoints\": [{\"group_name\": \"default_group\", \"endpoint\": \"\"}]}"


func TestFixedMethodPrice_initPricingData(t *testing.T) {
	metadata,_ := blockchain.InitServiceMetaDataFromJson(testJsonDataFixedMethodPrice)
	grpcCtx := &handler.GrpcStreamContext{Info:&grpc.StreamServerInfo{FullMethod:"/example_service.Calculator/add"}}
	pricing,_ := InitPricingStrategy(metadata)
	price,err := pricing.GetPrice(grpcCtx)
	assert.Equal(t,price,big.NewInt(2))
	assert.Nil(t,err)
	//Test with an undefined method Name
	grpcCtx.Info.FullMethod= "NonDefinedMethod"
	price,err = pricing.GetPrice(grpcCtx)
	assert.Nil(t,price)
	assert.Equal(t,err.Error(),"price is not defined for the Method NonDefinedMethod")
	//Test if the metadata is not properly defined
	metadata.Pricing.Details = nil

	pricing,err = InitPricingStrategy(metadata)
	assert.Equal(t,err.Error(),"service / method level pricing is not defined correctly")
	assert.Nil(t,pricing)

}


