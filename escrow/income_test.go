package escrow

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

type incomeValidatorMockType struct {
	err error
}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeData) (err error) {
	return incomeValidator.err
}

func TestIncomeValidate(t *testing.T) {
	one := big.NewInt(1)
	income := big.NewInt(0)
	incomeValidator := NewIncomeValidator(big.NewInt(0))
	price := big.NewInt(0)

	income.Sub(price, one)
	err := incomeValidator.Validate(&IncomeData{Income: income})
	msg := fmt.Sprintf("income %s does not equal to price %s", income, price)
	assert.Equal(t, NewPaymentError(Unauthenticated, msg), err)

	income.Set(price)
	err = incomeValidator.Validate(&IncomeData{Income: income})
	assert.Nil(t, err)

	income.Add(price, one)
	err = incomeValidator.Validate(&IncomeData{Income: income})
	msg = fmt.Sprintf("income %s does not equal to price %s", income, price)
	assert.Equal(t, NewPaymentError(Unauthenticated, msg), err)
}
