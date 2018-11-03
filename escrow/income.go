package escrow

import (
	"math/big"

	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	// Validate returns nil if validation is successful or correct gRPC status
	// to be sent to client in case of validation error.
	Validate(*IncomeData) (err *status.Status)
}

type incomeValidator struct {
}

// NewIncomeValidator returns new income validator instance
func NewIncomeValidator() (validator IncomeValidator) {
	return &incomeValidator{}
}

func (validator *incomeValidator) Validate(data *IncomeData) (err *status.Status) {

	price := config.GetBigInt(config.PricePerCallKey)

	if data.Income.Cmp(price) != 0 {
		err = status.Newf(codes.Unauthenticated, "income %d does not equal to price %d", data.Income, price)
		return
	}

	return
}
