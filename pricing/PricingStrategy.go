package pricing

import (
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
)

type PricingStrategy struct {
	//Holds all the pricing types possible
	pricingTypes []PriceType
}

//Figure out which price type is to be used
func (pricing PricingStrategy) determinePricingApplicable(GrpcContext *handler.GrpcStreamContext) (priceType PriceType, err error) {
	//For future , there could be multiple pricingTypes to select from and this method will help decide which pricing to pick
	//but for now , we just have one pricing Type ( either Fixed Price or Fixed price per Method)
	return pricing.pricingTypes[0], nil
}

//Initialize all the pricing types
func InitPricingStrategy(metadata *blockchain.ServiceMetadata) (*PricingStrategy, error) {
	pricing := &PricingStrategy{}

	if err := pricing.initFromMetaData(metadata); err != nil {
		log.WithError(err)
		return nil, err
	}
	return pricing, nil
}

func (pricing *PricingStrategy) AddPricingTypes(priceType PriceType)  {
	if pricing.pricingTypes == nil {
		pricing.pricingTypes = make([]PriceType, 0)
	}
	pricing.pricingTypes = append(pricing.pricingTypes, priceType)
}

func (pricing PricingStrategy) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	//Based on the input request , determine which price type is to be used
	if priceType, err := pricing.determinePricingApplicable(GrpcContext); err != nil {
		return nil, err
	} else {
		return priceType.GetPrice(GrpcContext)
	}
}

//Set all the PricingStrategy Types in this method.
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

	if priceType == nil  {
		err = fmt.Errorf("No PricingStrategy strategy defined in Metadata ")
		log.WithError(err)
	}
	return err

}
