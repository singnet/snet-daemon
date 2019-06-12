package price

import (
	"github.com/singnet/snet-daemon/blockchain"
	"google.golang.org/grpc"
	"math/big"
)

type FixedMethodPrice struct {
	//Service/Method is the key and value is the price
	priceInCogs *big.Int;
}


func (priceType FixedMethodPrice)GetPrice(ss grpc.ServerStream) (price *big.Int , err error) {
	return priceType.priceInCogs,nil
}

func (priceType FixedMethodPrice)initPricingData() {
	priceType.priceInCogs = blockchain.ServiceMetaData().GetPriceInCogs()
}