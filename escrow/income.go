package escrow

import (
	"github.com/singnet/snet-daemon/price"
	"math/big"

	"github.com/singnet/snet-daemon/handler"
)

// IncomeData is used to pass information to the pricing validation system.
// This system can use information about call to calculate price and verify
// income received.
type IncomeData struct {
	// Income is a difference between previous authorized amount and amount
	// which was received with current call.
	Income *big.Int
	// GrpcContext contains gRPC stream context information. For instance
	// metadata could be used to pass invoice id to check pricing.
	GrpcContext *handler.GrpcStreamContext
}

// IncomeValidator uses pricing information to check that call was payed
// correctly by channel sender. This interface can be implemented differently
// depending on pricing policy. For instance one can verify that call is payed
// according to invoice. Each RPC method can have different price and so on. To
// implement this strategies additional information from gRPC context can be
// required. In such case it should be added into handler.GrpcStreamContext.
type IncomeValidator interface {
	// Validate returns nil if validation is successful or correct PaymentError
	// status to be sent to client in case of validation error.
	Validate(*IncomeData) (err error)
}

type incomeValidator struct {
	priceStrategy *price.Pricing
}

// NewIncomeValidator returns new income validator instance
func NewIncomeValidator(pricing *price.Pricing) (validator IncomeValidator) {
	return &incomeValidator{priceStrategy: pricing}
}

func (validator *incomeValidator) Validate(data *IncomeData) (err error) {
//TO DO, the user request information from IncomeData needs to be passed here !!!!
	price,_ := validator.priceStrategy.GetPrice(data.GrpcContext)

	if data.Income.Cmp(price) != 0 {
		err = NewPaymentError(Unauthenticated, "income %d does not equal to price %d", data.Income, price)
		return
	}

	return
}
