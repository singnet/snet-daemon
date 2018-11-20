package escrow

import (
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"testing"
)

type incomeValidatorMockType struct {
	err *status.Status
}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeData) (err *status.Status) {
	return incomeValidator.err
}

func TestIncomeValidate(t *testing.T) {

	//price := config.GetBigInt(config.PricePerCallKey)
	blockchain.SetServiceMetaDataThroughJSON("{   \"version\": 1,   \"display_name\": \"ExampleService\",   \"encoding\": \"json\",   \"service_type\": \"jsonrpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"QmVPJ9KgvMRR28tH1p34TAuD5c5rveQ3DdstRTGTnoyzfJ\",   \"mpe_address\": \"0x5C7a4290F6F8FF64c69eEffDFAFc8644A4Ec3a4E\",   \"pricing\": {     \"price_model\": \"fixed_price\",     \"price_in_cogs\": 10000   },   \"groups\": [     {       \"group_name\": \"group1\",       \"group_id\": \"R0Y35/cmMXgnB485kEFxYPwlzeYFg2khJx3u/Bw+SnU=\",       \"payment_address\": \"0x42A605c07EdE0E1f648aB054775D6D4E38496144\"     },     {       \"group_name\": \"group2\",       \"group_id\": \"ClWK+kUulalkKEKE4CA/zTlvZpiw2jVJM1864efwTfU=\",       \"payment_address\": \"0x0067b427E299Eb2A4CBafc0B04C723F77c6d8a18\"     }   ],   \"endpoints\": [     {       \"group_name\": \"group1\",       \"endpoint\": \"8.8.8.8:2020\"     },     {       \"group_name\": \"group1\",       \"endpoint\": \"9.8.9.8:8080\"     },     {       \"group_name\": \"group2\",       \"endpoint\": \"8.8.8.8:22\"     },     {       \"group_name\": \"group2\",       \"endpoint\": \"1.2.3.4:8080\"     }   ] }")
	price := blockchain.GetPriceinCogs()
	assert.True(t, price.Cmp(big.NewInt(0)) > 0, "Invalid price_per_call value in default config", price)

	one := big.NewInt(1)
	income := big.NewInt(0)
	incomeValidator := NewIncomeValidator()

	income.Sub(price, one)
	err := incomeValidator.Validate(&IncomeData{Income: income})
	msg := fmt.Sprintf("income %s does not equal to price %s", income, price)
	assert.Equal(t, status.New(codes.Unauthenticated, msg), err)

	income.Set(price)
	err = incomeValidator.Validate(&IncomeData{Income: income})
	assert.Nil(t, err)

	income.Add(price, one)
	err = incomeValidator.Validate(&IncomeData{Income: income})
	msg = fmt.Sprintf("income %s does not equal to price %s", income, price)
	assert.Equal(t, status.New(codes.Unauthenticated, msg), err)
}
