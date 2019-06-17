package price

import (
	"github.com/singnet/snet-daemon/handler"
	"math/big"
)

const (

	 FIXED_METHOD_PRICING ="fixed_price_per_method";
	 FIXED_PRICING="fixed_price"
)
//Based on the request passed, a particular strategy will be picked up for processing
type iPrice interface {

	//Based on the user input determine how will the price be determined
	GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int , err error)
}

