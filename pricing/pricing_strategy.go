package pricing

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"

	"go.uber.org/zap"
)

type PricingStrategy struct {
	//Holds all the pricing types possible
	pricingTypes    map[string]PriceType
	serviceMetaData *blockchain.ServiceMetadata
}

// Figure out which price type is to be used
func (pricing PricingStrategy) determinePricingApplicable(fullMethod string) (priceType PriceType, err error) {
	//For future, there could be multiple pricingTypes to select from and this method will help decide which pricing to pick
	//but for now, we just have one pricing Type ( either Fixed Price or Fixed price per Method)

	if config.GetBool(config.EnableDynamicPricing) {
		//Use Dynamic pricing ONLY when you find the mapped price method to be called.
		if _, ok := pricing.serviceMetaData.GetDynamicPricingMethodAssociated(fullMethod); ok {
			return pricing.pricingTypes[DYNAMIC_PRICING], nil
		} else {
			zap.L().Info("No Dynamic Price method defined in service proto", zap.String("Method", fullMethod))
		}
	}
	//Default pricing is Fixed Pricing
	return pricing.pricingTypes[pricing.serviceMetaData.GetDefaultPricing().PriceModel], nil
}

// Initialize all the pricing types
func InitPricingStrategy(metadata *blockchain.ServiceMetadata) (*PricingStrategy, error) {
	pricing := &PricingStrategy{serviceMetaData: metadata}

	if err := pricing.initFromMetaData(metadata); err != nil {
		zap.L().Error(err.Error())
		return nil, err
	}
	return pricing, nil
}

func (pricing *PricingStrategy) AddPricingTypes(priceType PriceType) {
	if pricing.pricingTypes == nil {
		pricing.pricingTypes = make(map[string]PriceType)
	}
	pricing.pricingTypes[priceType.GetPriceType()] = priceType
}

func (pricing PricingStrategy) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	//Based on the input request , determine which price type is to be used
	if priceType, err := pricing.determinePricingApplicable(GrpcContext.Info.FullMethod); err != nil {
		return nil, err
	} else {
		return priceType.GetPrice(GrpcContext)
	}
}

// Set all the PricingStrategy Types in this method.
func (pricing *PricingStrategy) initFromMetaData(metadata *blockchain.ServiceMetadata) (err error) {
	var priceType PriceType

	if strings.Compare(metadata.GetDefaultPricing().PriceModel, FIXED_PRICING) == 0 {
		priceType = &FixedPrice{priceInCogs: metadata.GetDefaultPricing().PriceInCogs}

	} else if strings.Compare(metadata.GetDefaultPricing().PriceModel, FIXED_METHOD_PRICING) == 0 {
		methodPricing := &FixedMethodPrice{}
		err = methodPricing.initPricingData(metadata)
		priceType = methodPricing
	}
	pricing.AddPricingTypes(priceType)
	if config.GetBool(config.EnableDynamicPricing) {
		pricing.AddPricingTypes(&DynamicMethodPrice{
			serviceMetaData: metadata,
		})
	}
	if priceType == nil {
		err = fmt.Errorf("No PricingStrategy strategy defined in Metadata ")
		zap.L().Error(err.Error())
	}
	return err

}
