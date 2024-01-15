package blockchain

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/ipfsutils"
	log "github.com/sirupsen/logrus"
	"math/big"
	"os"
	"strings"
)

/*
	{
	    "version": 1,
	    "display_name": "Entity Disambiguation",
	    "encoding": "proto",
	    "service_type": "grpc",
	    "model_ipfs_hash": "Qmd21xqgX8fkU4fD2bFMNG2Q86wAB4GmGBekQfLoiLtXYv",
	    "mpe_address": "0x34E2EeE197EfAAbEcC495FdF3B1781a3b894eB5f",
	    "groups": [
	        {
	            "group_name": "default_group",
	            "free_calls": 12,
	            "free_call_signer_address": "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F",
	            "pricing": [
	                {
	                    "price_model": "fixed_price",
	                    "price_in_cogs": 1,
	                    "default": true
	                }
	            ],
	            "endpoints": [
	                "https://tz-services-1.snet.sh:8005"
	            ],
	            "group_id": "EoFmN3nvaXpf6ew8jJbIPVghE5NXfYupFF7PkRmVyGQ=",

	{
	  "licenses": {

	    "tiers": [{
	      "type": "Tier",
	      "planName": "Tier AAA",
	      "grpcServiceName": "ServiceA",
	      "grpcMethodName": "MethodA",
	      "range": [
	        {
	          "high": 100,
	          "DiscountInPercentage": 3
	        },
	        {
	          "high": 200,
	          "DiscountInPercentage": 4
	        },
	        {
	          "high": 300,
	          "DiscountInPercentage": 6
	        }
	      ],
	      "detailsUrl": "http://abc.org/licenses/Tier.html",
	      "isActive": "true/false"
	    },
	     {
	      "type": "Tier",
	      "planName": "Tier BBB Applicable for All service.methods",
	      "range": [
	        {
	          "high": 100,
	          "DiscountInPercentage": 1
	        },
	        {
	          "high": 200,
	          "DiscountInPercentage": 1.75
	        },
	        {
	          "high": 300,
	          "DiscountInPercentage": 2.50
	        }
	      ],
	      "detailsUrl": "http://abc.org/licenses/Tier.html",
	      "isActive": "true/false"
	    }],
	    "subscriptions": {
	      "type": "Subscription",
	      "subscription": [
	        {
	          "periodInDays": 30,
	          "DiscountInPercentage": 10,
	          "planName": "Monthly For ServiceA/MethodA",
	          "LicenseCost": 90,
	          "grpcServiceName": "ServiceA",
	          "grpcMethodName": "MethodA"
	        },
	        {
	          "periodInDays": 30,
	          "DiscountInPercentage": 12,
	          "planName": "Monthly",
	          "LicenseCost": 93
	        },
	        {
	          "periodInDays": 120,
	          "DiscountInPercentage": 16,
	          "LicenseCost": 120,
	          "planName": "Quarterly"
	        },
	        {
	          "periodInDays": 365,
	          "DiscountInPercentage": 23,
	          "LicenseCost": 390,
	          "planName": "Yearly"
	        }
	      ],
	      "detailsUrl": "http://abc.org/licenses/Subscription.html",
	      "isActive": "true/false"
	    }
	  }
	}

	    ],
	    "assets": {
	        "hero_image": "Qmb1n3LxPXLHTUMu7afrpZdpug4WhhcmVVCEwUxjLQafq1/hero_named-entity-disambiguation.png"
	    },
	    "service_description": {
	        "url": "https://singnet.github.io/nlp-services-misc/users_guide/named-entity-disambiguation-service.html",
	        "description": "Provide further clearity regaridng entities named within a piece of text. For example, \"Paris is the capital of France\", we would want to link \"Paris\" to Paris the city not Paris Hilton in this case.",
	        "short_description": "text of 180 chars"
	    },
	    "contributors": [
	            {
	                "name": "dummy dummy",
	                "email_id": "dummy@dummy.io"
	            }
	        ]
	}
*/
const IpfsPrefix = "ipfs://"

type ServiceMetadata struct {
	Version       int                 `json:"version"`
	DisplayName   string              `json:"display_name"`
	Encoding      string              `json:"encoding"`
	ServiceType   string              `json:"service_type"`
	Groups        []OrganizationGroup `json:"groups"`
	ModelIpfsHash string              `json:"model_ipfs_hash"`
	MpeAddress    string              `json:"mpe_address"`

	multiPartyEscrowAddress common.Address
	defaultPricing          Pricing

	defaultGroup OrganizationGroup

	freeCallSignerAddress     common.Address
	isfreeCallAllowed         bool
	freeCallsAllowed          int
	dynamicPriceMethodMapping map[string]string `json:"dynamicpricing"`
	trainingMethods           []string          `json:"training_methods"`
}
type Tiers struct {
	Tiers Tier `json:"tier"`
}
type AddOns struct {
	DiscountInPercentage float64 `json:"discountInPercentage"`
	AddOnCostInAGIX      int     `json:"addOnCostInAGIX"`
	Name                 string  `json:"name"`
}
type TierRange struct {
	High                 int     `json:"high"`
	DiscountInPercentage float64 `json:"DiscountInPercentage"`
}

type Subscription struct {
	PeriodInDays         int     `json:"periodInDays"`
	DiscountInPercentage float64 `json:"discountInPercentage"`
	PlanName             string  `json:"planName"`
	LicenseCost          big.Int `json:"licenseCost"`
	GrpcServiceName      string  `json:"grpcServiceName,omitempty"`
	GrpcMethodName       string  `json:"grpcMethodName,omitempty"`
}

type Subscriptions struct {
	Type         string         `json:"type"`
	DetailsURL   string         `json:"detailsUrl"`
	IsActive     string         `json:"isActive"`
	Subscription []Subscription `json:"subscription"`
}
type Tier struct {
	Type            string      `json:"type"`
	PlanName        string      `json:"planName"`
	GrpcServiceName string      `json:"grpcServiceName,omitempty"`
	GrpcMethodName  string      `json:"grpcMethodName,omitempty"`
	Range           []TierRange `json:"range"`
	DetailsURL      string      `json:"detailsUrl"`
	IsActive        string      `json:"isActive"`
}
type Licenses struct {
	Subscriptions Subscriptions `json:"subscriptions,omitempty"`
	Tiers         []Tier        `json:"tiers"`
}

type OrganizationGroup struct {
	Endpoints      []string  `json:"endpoints"`
	GroupID        string    `json:"group_id"`
	GroupName      string    `json:"group_name"`
	Pricing        []Pricing `json:"pricing"`
	FreeCalls      int       `json:"free_calls"`
	FreeCallSigner string    `json:"free_call_signer_address"`
	Licenses       Licenses  `json:"licenses,omitempty"`
	AddOns         []AddOns  `json:"addOns,omitempty"`
}
type Pricing struct {
	PriceModel     string           `json:"price_model"`
	PriceInCogs    *big.Int         `json:"price_in_cogs,omitempty"`
	PackageName    string           `json:"package_name,omitempty"`
	Default        bool             `json:"default,omitempty"`
	PricingDetails []PricingDetails `json:"details,omitempty"`
}

type PricingDetails struct {
	ServiceName   string          `json:"service_name"`
	MethodPricing []MethodPricing `json:"method_pricing"`
}
type MethodPricing struct {
	MethodName  string   `json:"method_name"`
	PriceInCogs *big.Int `json:"price_in_cogs"`
}

func getRegistryAddressKey() common.Address {
	address := config.GetRegistryAddress()
	return common.HexToAddress(address)
}

func (metaData ServiceMetadata) GetDefaultPricing() Pricing {
	return metaData.defaultPricing
}

func ServiceMetaData() *ServiceMetadata {
	var metadata *ServiceMetadata
	var err error
	if config.GetBool(config.BlockchainEnabledKey) {
		ipfsHash := string(getServiceMetaDataUrifromRegistry())
		metadata, err = GetServiceMetaDataFromIPFS(FormatHash(ipfsHash))
		if err != nil {
			log.WithError(err).
				Panic("error on determining service metadata from file")
		}
	} else {
		metadata = &ServiceMetadata{Encoding: "proto", ServiceType: "grpc"}
	}
	return metadata
}

func ReadServiceMetaDataFromLocalFile(filename string) (*ServiceMetadata, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file: %v", filename)
	}
	strJson := string(file)
	metadata, err := InitServiceMetaDataFromJson(strJson)

	if err != nil {
		return nil, fmt.Errorf("error reading local file service_metadata.json ")
	}
	return metadata, nil
}

func getRegistryCaller() (reg *RegistryCaller) {
	ethClient, err := GetEthereumClient()
	if err != nil {

		log.WithError(err).
			Panic("Unable to get Blockchain client ")

	}
	defer ethClient.Close()
	registryContractAddress := getRegistryAddressKey()
	reg, err = NewRegistryCaller(registryContractAddress, ethClient.EthClient)
	if err != nil {
		log.WithError(err).WithField("registryContractAddress", registryContractAddress).
			Panic("Error instantiating Registry contract for the given Contract Address")
	}
	return reg
}

func getServiceMetaDataUrifromRegistry() []byte {
	reg := getRegistryCaller()

	orgId := StringToBytes32(config.GetString(config.OrganizationId))
	serviceId := StringToBytes32(config.GetString(config.ServiceId))

	serviceRegistration, err := reg.GetServiceRegistrationById(nil, orgId, serviceId)
	if err != nil || !serviceRegistration.Found {
		log.WithError(err).WithField("OrganizationId", config.GetString(config.OrganizationId)).
			WithField("ServiceId", config.GetString(config.ServiceId)).
			Panic("Error Retrieving contract details for the Given Organization and Service Ids ")
	}

	return serviceRegistration.MetadataURI[:]
}

func GetServiceMetaDataFromIPFS(hash string) (*ServiceMetadata, error) {
	jsondata := ipfsutils.GetIpfsFile(hash)
	return InitServiceMetaDataFromJson(jsondata)
}

func InitServiceMetaDataFromJson(jsonData string) (*ServiceMetadata, error) {
	metaData := new(ServiceMetadata)
	err := json.Unmarshal([]byte(jsonData), &metaData)
	if err != nil {
		log.WithError(err).WithField("jsondata", jsonData)
		return nil, err
	}

	if err := setDerivedFields(metaData); err != nil {
		return nil, err
	}
	if err := setFreeCallData(metaData); err != nil {
		return nil, err
	}
	//If Dynamic pricing is enabled ,there will be mandatory checks on the service proto
	//this is to ensure that the standards on how one defines the methods to invoke is followed
	if config.GetBool(config.EnableDynamicPricing) {
		if err := setServiceProto(metaData); err != nil {
			return nil, err
		}
	}
	e, err := json.Marshal(metaData.dynamicPriceMethodMapping)
	if err != nil {
		log.Println(err)
	}

	log.Println(string(e))
	e1, err := json.Marshal(metaData.trainingMethods)
	if err != nil {
		log.Println(err)
	}

	log.Println(string(e1))
	return metaData, err
}

func setDerivedFields(metaData *ServiceMetadata) (err error) {
	if err = setDefaultPricing(metaData); err != nil {
		return err
	}
	setMultiPartyEscrowAddress(metaData)

	return nil

}

func setGroup(metaData *ServiceMetadata) (err error) {
	groupName := config.GetString(config.DaemonGroupName)
	for _, group := range metaData.Groups {
		if strings.Compare(group.GroupName, groupName) == 0 {
			metaData.defaultGroup = group
			return nil
		}
	}
	err = fmt.Errorf("group name %v in config is invalid, "+
		"there was no group found with this name in the metadata", groupName)
	log.WithError(err)
	return err
}

func setDefaultPricing(metaData *ServiceMetadata) (err error) {
	if err = setGroup(metaData); err != nil {
		return err
	}
	for _, pricing := range metaData.defaultGroup.Pricing {
		if pricing.Default {
			metaData.defaultPricing = pricing
			return nil
		}
	}
	err = fmt.Errorf("MetaData does not have the default pricing set ")
	log.WithError(err)
	return err
}

func setMultiPartyEscrowAddress(metaData *ServiceMetadata) {
	metaData.MpeAddress = ToChecksumAddress(metaData.MpeAddress)
	metaData.multiPartyEscrowAddress = common.HexToAddress(metaData.MpeAddress)
}

func setFreeCallData(metaData *ServiceMetadata) error {

	if metaData.defaultGroup.FreeCalls > 0 {
		metaData.isfreeCallAllowed = true
		metaData.freeCallsAllowed = metaData.defaultGroup.FreeCalls
		//If the signer address is not a valid address, then return back an error
		if !common.IsHexAddress(metaData.defaultGroup.FreeCallSigner) {
			return fmt.Errorf("MetaData does not have 'free_call_signer_address defined correctly")
		}
		metaData.freeCallSignerAddress = common.HexToAddress(ToChecksumAddress(metaData.defaultGroup.FreeCallSigner))
	}
	return nil
}

func (metaData *ServiceMetadata) GetMpeAddress() common.Address {
	return metaData.multiPartyEscrowAddress
}

func (metaData *ServiceMetadata) FreeCallSignerAddress() common.Address {
	return metaData.freeCallSignerAddress
}

func (metaData *ServiceMetadata) GetWireEncoding() string {
	return metaData.Encoding
}

func (metaData *ServiceMetadata) GetVersion() int {
	return metaData.Version
}

func (metaData *ServiceMetadata) GetServiceType() string {
	return metaData.ServiceType
}

func (metaData *ServiceMetadata) GetDisplayName() string {
	return metaData.DisplayName
}

func (metaData *ServiceMetadata) IsFreeCallAllowed() bool {
	return metaData.isfreeCallAllowed
}

func (metaData *ServiceMetadata) GetFreeCallsAllowed() int {
	return metaData.freeCallsAllowed
}

func (metaData *ServiceMetadata) GetLicenses() Licenses {
	return metaData.defaultGroup.Licenses
}

// methodFullName , ex "/example_service.Calculator/add"
func (metaData *ServiceMetadata) GetDynamicPricingMethodAssociated(methodFullName string) (pricingMethod string, isDynamicPricingEligible bool) {
	//Check if Method Level Options are defined , for the given Service and method,
	//If Defined check if its in the format supported , then return the full method Name
	// i.e /package.service/method format , this will be directly fed in to the grpc called to made to
	//determine dynamic pricing
	if !config.GetBool(config.EnableDynamicPricing) {
		return
	}
	pricingMethod = metaData.dynamicPriceMethodMapping[methodFullName]
	if strings.Compare("", pricingMethod) == 0 {
		isDynamicPricingEligible = false
	} else {
		isDynamicPricingEligible = true
	}
	return
}

// methodFullName , ex "/example_service.Calculator/add"
func (metaData *ServiceMetadata) IsModelTraining(methodFullName string) (useModelTrainingEndPoint bool) {

	if !config.GetBool(config.ModelTrainingEnabled) {
		return false
	}
	useModelTrainingEndPoint = isElementInArray(methodFullName, metaData.trainingMethods)
	return
}
func isElementInArray(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
func setServiceProto(metaData *ServiceMetadata) (err error) {
	metaData.dynamicPriceMethodMapping = make(map[string]string, 0)
	metaData.trainingMethods = make([]string, 0)
	//This is to handler the scenario where there could be multiple protos associated with the service proto
	protoFiles, err := ipfsutils.ReadFilesCompressed(ipfsutils.GetIpfsFile(metaData.ModelIpfsHash))
	for _, file := range protoFiles {
		if srvProto, err := parseServiceProto(file); err != nil {
			return err
		} else {
			dynamicMethodMap, trainingMethodMap, err := buildDynamicPricingMethodsMap(srvProto)
			if err != nil {
				return err
			}
			metaData.dynamicPriceMethodMapping = dynamicMethodMap
			metaData.trainingMethods = trainingMethodMap
		}
	}

	return nil
}

func parseServiceProto(serviceProtoFile string) (*proto.Proto, error) {
	reader := strings.NewReader(serviceProtoFile)
	parser := proto.NewParser(reader)
	parsedProto, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return parsedProto, nil
}

func buildDynamicPricingMethodsMap(serviceProto *proto.Proto) (dynamicPricingMethodMapping map[string]string,
	trainingMethodPricing []string, err error) {
	dynamicPricingMethodMapping = make(map[string]string, 0)
	trainingMethodPricing = make([]string, 0)
	var pkgName, serviceName, methodName string
	for _, elem := range serviceProto.Elements {
		//package is parsed earlier than service ( per documentation)
		if pkg, ok := elem.(*proto.Package); ok {
			pkgName = pkg.Name
		}

		if service, ok := elem.(*proto.Service); ok {
			serviceName = service.Name
			for _, serviceElements := range service.Elements {
				if rpcMethod, ok := serviceElements.(*proto.RPC); ok {
					methodName = rpcMethod.Name
					for _, methodOption := range rpcMethod.Options {
						if strings.Compare(methodOption.Name, "(pricing.my_method_option).estimatePriceMethod") == 0 {
							pricingMethod := fmt.Sprintf("%v", methodOption.Constant.Source)
							dynamicPricingMethodMapping["/"+pkgName+"."+serviceName+"/"+methodName+""] =
								pricingMethod
						}
						if strings.Compare(methodOption.Name, "(training.my_method_option).trainingMethodIndicator") == 0 {
							trainingMethod := "/" + pkgName + "." + serviceName + "/" + methodName + ""
							trainingMethodPricing = append(trainingMethodPricing, trainingMethod)
						}
					}
				}
			}
		}
	}
	return
}
