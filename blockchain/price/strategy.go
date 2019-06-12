package price

import (
	"github.com/singnet/snet-daemon/blockchain"
	"google.golang.org/grpc"
	"math/big"
)


type PricingStrategy struct {
	pricingTypes []iPriceType
}

//Figure out which price type is to be used
func determinePricingStrategy(ss grpc.ServerStream) (priceType iPriceType,err error) {
 	return nil,nil
}

func InitStrategy(metadata *blockchain.ServiceMetadata) *PricingStrategy {
	strategy := &PricingStrategy{}
	//Set all the Pricing Types here
	strategy.pricingTypes = make([]iPriceType,0)
	return strategy
}

func (strategy PricingStrategy) GetPrice(ss grpc.ServerStream) (price *big.Int , err error) {
	//Based on the input request , determine which price type is to be used
	priceType,err := determinePricingStrategy(ss)
	return priceType.getPrice(ss)
}