package price

import (
	"google.golang.org/grpc"
	"math/big"
)

//Based on the request passed, a particular strategy will be picked up for processing
type iPriceType interface {
	//Every Pricing strategy needs to determine the source of your pricing
	initPricingData()
	//Based on the user input determine how will the price be determined
	getPrice(ss grpc.ServerStream) (price *big.Int , err error)
}

