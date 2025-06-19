package pricing

import (
	"github.com/singnet/snet-daemon/v6/handler"
	"math/big"
)

type FixedPrice struct {
	//Value initialized from metaData
	priceInCogs *big.Int
}

func (priceType FixedPrice) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	return priceType.priceInCogs, nil
}

func (priceType FixedPrice) GetPriceType() string {
	return FIXED_PRICING
}
