package price

import (
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
	"math/big"
)

type FixedMethodPrice struct {
	//Service/Method is the key and value is he price
	methodToPriceMap map[string]*big.Int
}

func (priceType FixedMethodPrice) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	//The returned string is in the format of "/packagename.service/method", for example /example_service.Calculator/mul
	methodName := GrpcContext.Info.FullMethod
	if price, ok := priceType.methodToPriceMap[methodName]; !ok {
		return nil, fmt.Errorf("price is not defined for the Method %v", methodName)
	} else {
		return price, nil
	}
}

func (priceType *FixedMethodPrice) initPricingData(metadata *blockchain.ServiceMetadata) (err error) {
	priceType.methodToPriceMap = make(map[string]*big.Int)
	prefix:= metadata.Pricing.PackageName
	if len(prefix)>0 {
		prefix=prefix+"."
	}
	for _, detail := range metadata.Pricing.Details {
		for _, methodDetails := range detail.MethodPricing {
			priceType.methodToPriceMap["/"+prefix+detail.ServiceName+"/"+methodDetails.MethodName] = methodDetails.PriceInCogs
		}
	}
	if len(priceType.methodToPriceMap) == 0 {
		return fmt.Errorf("service / method level pricing Not defined correctly")
	}

	return nil
}

