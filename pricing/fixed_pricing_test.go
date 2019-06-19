package pricing

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/handler"
)

var testJsonDataFixedPrice = "{\"version\": 1, \"display_name\": \"Example1\", \"encoding\": \"grpc\", \"service_type\": \"grpc\", \"payment_expiration_threshold\": 40320, \"model_ipfs_hash\": \"QmQC9EoVdXRWmg8qm25Hkj4fG79YAgpNJCMDoCnknZ6VeJ\", \"mpe_address\": \"0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E\", \"pricing\": {\"price_model\": \"fixed_price\", \"price_in_cogs\": 12000000}, \"groups\": [{\"group_name\": \"default_group\", \"group_id\": \"nXzNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U=\", \"payment_address\": \"0xD6C6344f1D122dC6f4C1782A4622B683b9008081\"}], \"endpoints\": [{\"group_name\": \"default_group\", \"endpoint\": \"\"}]}"

func TestFixedPrice_GetPrice(t *testing.T) {
	metadata, _ := blockchain.InitServiceMetaDataFromJson(testJsonDataFixedPrice)
	grpcCtx := &handler.GrpcStreamContext{}
	pricing,_ := InitPricingStrategy(metadata)

	price,err := pricing.pricingTypes[0].GetPrice(grpcCtx)
	assert.Equal(t,price,big.NewInt(12000000))
	assert.Nil(t,err)
}
