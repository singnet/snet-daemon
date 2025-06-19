package pricing

import (
	"github.com/singnet/snet-daemon/v6/handler"
	"math/big"
)

const (
	FIXED_METHOD_PRICING = "fixed_price_per_method"
	FIXED_PRICING        = "fixed_price"
	DYNAMIC_PRICING      = "dynamic_pricing"
)

// Based on the request passed, a particular strategy will be picked up for processing
type PriceType interface {

	//Based on the user input determine how will the price be determined
	GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error)
	//Will tell the price type the instance is associated with
	GetPriceType() string
}
