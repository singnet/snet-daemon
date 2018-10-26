package escrow

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIncomeValidate(t *testing.T) {

	price := config.GetBigInt(config.PricePerCallKey)
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
