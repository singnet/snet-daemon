package price

import (
	"github.com/singnet/snet-daemon/handler"
	"math/big"
)

type FixedPrice struct {
	//Value initialized from metaData
	priceInCogs *big.Int;
}

func (priceType FixedPrice) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int , err error) {
	return priceType.priceInCogs,nil
}
