package price

import (
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/handler"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
)

type Pricing struct {
	//Holds all the pricing types possible
	pricingTypes []iPrice
}

//Figure out which price type is to be used
func (pricing Pricing) determinePricingApplicable(GrpcContext *handler.GrpcStreamContext) (priceType iPrice, err error) {
	//For future , there could be multiple pricingTypes to select from and this method will help decide which pricing to pick
	//but for now , we just have one pricing Type ( either Fixed Price or Fixed price per Method)
	return pricing.pricingTypes[0], nil
}

//Initialize all the pricing types
func InitPricing(metadata *blockchain.ServiceMetadata) (*Pricing, error) {
	pricing := &Pricing{}

	if err := pricing.initFromMetaData(metadata); err != nil {
		log.WithError(err)
		return nil, err
	}
	return pricing, nil
}

func (pricing *Pricing) AddPricingTypes(priceType iPrice)  {
	if pricing.pricingTypes == nil {
		pricing.pricingTypes = make([]iPrice, 0)
	}
	pricing.pricingTypes = append(pricing.pricingTypes, priceType)
}

func (pricing Pricing) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	//Based on the input request , determine which price type is to be used
	if priceType, err := pricing.determinePricingApplicable(GrpcContext); err != nil {
		return nil, err
	} else {
		return priceType.GetPrice(GrpcContext)
	}
}

//Set all the Pricing Types in this method.
func (pricing *Pricing) initFromMetaData(metadata *blockchain.ServiceMetadata) (err error) {
	var priceType iPrice

	if strings.Compare(metadata.Pricing.PriceModel, FIXED_PRICING) == 0 {
		priceType = &FixedPrice{priceInCogs: metadata.Pricing.PriceInCogs}

	} else if strings.Compare(metadata.Pricing.PriceModel, FIXED_METHOD_PRICING) == 0 {
		methodPricing := &FixedMethodPrice{}
		err = methodPricing.initPricingData(metadata)
		priceType = methodPricing
	}
	pricing.AddPricingTypes(priceType)

	if len(pricing.pricingTypes) == 0 {
		err = fmt.Errorf("No Pricing strategy defined in Metadata ")
		log.WithError(err)
	}
	return err

}
