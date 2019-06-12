package price

import (
	"fmt"
	"google.golang.org/grpc"
	"math/big"
)

type FixedPrice struct {
	//Service/Method is the key and value is he price
	methodToPriceMap map[string]*big.Int;
}


func (priceType FixedPrice)GetPrice(ss grpc.ServerStream) (price *big.Int , err error) {
	// The returned string is in the format of "/service/method".
	methodName, _ := grpc.MethodFromServerStream(ss)
	if price,ok := priceType.methodToPriceMap[methodName]; !ok {
		return nil,fmt.Errorf("iPriceType Not defined for the Method %v",methodName)
	} else {
		return price ,nil
	}
}

func (priceType FixedPrice)initPricingData() {
	priceType.methodToPriceMap = make(map[string]*big.Int)
}
