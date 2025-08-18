package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"os"
	"slices"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/singnet/snet-daemon/v6/errs"
	"github.com/singnet/snet-daemon/v6/utils"

	pproto "github.com/emicklei/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/ipfsutils"
	"go.uber.org/zap"
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

type ServiceMetadata struct {
	Version          int                 `json:"version"`
	DisplayName      string              `json:"display_name"`
	Encoding         string              `json:"encoding"`
	ServiceType      string              `json:"service_type"`
	Groups           []OrganizationGroup `json:"groups"`
	ModelIpfsHash    string              `json:"model_ipfs_hash"`
	ServiceApiSource string              `json:"service_api_source"`
	MpeAddress       string              `json:"mpe_address"`

	multiPartyEscrowAddress common.Address
	defaultPricing          Pricing

	defaultGroup OrganizationGroup

	freeCallSignerAddress     common.Address
	isfreeCallAllowed         bool
	freeCallsAllowed          int
	DynamicPriceMethodMapping map[string]string `json:"dynamic_pricing"`
	TrainingMethods           []string          `json:"training_methods"`
	TrainingMetadata          map[string]any    `json:"training_metadata"`
	ProtoDescriptors          linker.Files      `json:"-"`
	ProtoFiles                map[string]string `json:"-"`
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

func ServiceMetaData() *ServiceMetadata {
	var metadata *ServiceMetadata
	var err error
	var ipfsHash []byte
	if !config.GetBool(config.BlockchainEnabledKey) {
		metadata = &ServiceMetadata{Encoding: "proto", ServiceType: "grpc"}
		return metadata
	}
	ipfsHash, err = getServiceMetaDataURIFromRegistry()
	if err != nil {
		zap.L().Fatal(err.Error()+errs.ErrDescURL(errs.InvalidConfig),
			zap.String("OrganizationId", config.GetString(config.OrganizationId)),
			zap.String("ServiceId", config.GetString(config.ServiceId)))
	}
	metadata, err = GetServiceMetaDataFromIPFS(string(ipfsHash))
	if err != nil {
		zap.L().Panic("error on determining service metadata from file"+errs.ErrDescURL(errs.InvalidMetadata), zap.Error(err))
	}

	zap.L().Info("service_type", zap.String("value", metadata.GetServiceType()))
	return metadata
}

func ReadServiceMetaDataFromLocalFile(filename string) (*ServiceMetadata, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file: %v", filename)
	}
	metadata, err := InitServiceMetaDataFromJson(file)
	if err != nil {
		return nil, fmt.Errorf("error reading local file service_metadata.json ")
	}
	return metadata, nil
}

func getRegistryCaller() (reg *RegistryCaller) {
	ethHttpClient, err := CreateHTTPEthereumClient()
	if err != nil {
		zap.L().Panic("Unable to get Blockchain client ", zap.Error(err))
	}
	defer ethHttpClient.Close()
	registryContractAddress := getRegistryAddressKey()
	reg, err = NewRegistryCaller(registryContractAddress, ethHttpClient.EthClient)
	if err != nil {
		zap.L().Panic("Error instantiating Registry contract for the given Contract Address", zap.Error(err), zap.Any("registryContractAddress", registryContractAddress))
	}
	return reg
}

func GetRegistryFilterer(ethWsClient *ethclient.Client) *RegistryFilterer {
	registryContractAddress := getRegistryAddressKey()
	reg, err := NewRegistryFilterer(registryContractAddress, ethWsClient)
	if err != nil {
		zap.L().Panic("Error instantiating Registry contract for the given Contract Address", zap.Error(err), zap.Any("registryContractAddress", registryContractAddress))
	}
	return reg
}

func getServiceMetaDataURIFromRegistry() ([]byte, error) {
	reg := getRegistryCaller()

	orgId := utils.StringToBytes32(config.GetString(config.OrganizationId))
	serviceId := utils.StringToBytes32(config.GetString(config.ServiceId))

	serviceRegistration, err := reg.GetServiceRegistrationById(nil, orgId, serviceId)
	if err != nil || !serviceRegistration.Found {
		return nil, fmt.Errorf("error retrieving contract details for the given organization and service ids %v", err)
	}

	return serviceRegistration.MetadataURI[:], nil
}

func GetServiceMetaDataFromIPFS(hash string) (*ServiceMetadata, error) {
	jsondata, err := ipfsutils.ReadFile(hash)
	if err != nil {
		return nil, err
	}
	return InitServiceMetaDataFromJson(jsondata)
}

func InitServiceMetaDataFromJson(jsonData []byte) (*ServiceMetadata, error) {
	metaData := new(ServiceMetadata)
	err := json.Unmarshal(jsonData, &metaData)
	if err != nil {
		zap.L().Error(err.Error(), zap.Any("jsondata", jsonData))
		return nil, err
	}

	if err := metaData.setDerivedFields(); err != nil {
		return nil, err
	}
	if err := metaData.setFreeCallData(); err != nil {
		return nil, err
	}

	if err := metaData.setServiceProto(); err != nil {
		return nil, err
	}

	if len(metaData.DynamicPriceMethodMapping) > 0 {
		dynamicPriceMethodMappingJson, err := json.Marshal(metaData.DynamicPriceMethodMapping)
		if err != nil {
			zap.L().Error(err.Error())
		}
		zap.L().Debug("dynamic price method mapping", zap.String("json", string(dynamicPriceMethodMappingJson)))
	}

	return metaData, err
}

func (metaData *ServiceMetadata) GetDefaultPricing() Pricing {
	return metaData.defaultPricing
}

func (metaData *ServiceMetadata) setDerivedFields() (err error) {
	if err = metaData.setDefaultPricing(); err != nil {
		return err
	}
	metaData.setMultiPartyEscrowAddress()

	return nil
}

func (metaData *ServiceMetadata) setGroup() (err error) {
	groupName := config.GetString(config.DaemonGroupName)
	for _, group := range metaData.Groups {
		if strings.Compare(group.GroupName, groupName) == 0 {
			metaData.defaultGroup = group
			return nil
		}
	}
	err = fmt.Errorf("group name %v in config is invalid, "+
		"there was no group found with this name in the metadata", groupName)
	zap.L().Error("Error in set group", zap.Error(err))
	return err
}

func (metaData *ServiceMetadata) setDefaultPricing() (err error) {
	if err = metaData.setGroup(); err != nil {
		return err
	}
	for _, pricing := range metaData.defaultGroup.Pricing {
		if pricing.Default {
			metaData.defaultPricing = pricing
			return nil
		}
	}
	err = fmt.Errorf("metadata does not have the default pricing set")
	zap.L().Warn("[setDefaultPricing] Error in set default pricing", zap.Error(err))
	return err
}

func (metaData *ServiceMetadata) setMultiPartyEscrowAddress() {
	metaData.MpeAddress = utils.ToChecksumAddressStr(metaData.MpeAddress)
	metaData.multiPartyEscrowAddress = common.HexToAddress(metaData.MpeAddress)
}

func (metaData *ServiceMetadata) setFreeCallData() error {
	if metaData.defaultGroup.FreeCallSigner != "" {
		metaData.isfreeCallAllowed = true
		metaData.freeCallsAllowed = metaData.defaultGroup.FreeCalls
		// If the signer address is not a valid address, then return an error
		if !common.IsHexAddress(metaData.defaultGroup.FreeCallSigner) {
			return fmt.Errorf("metadata does not have 'free_call_signer_address defined correctly")
		}
		metaData.freeCallSignerAddress = utils.ToChecksumAddress(metaData.defaultGroup.FreeCallSigner)
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

// GetDynamicPricingMethodAssociated accept methodFullName, example "/example_service.Calculator/add"
func (metaData *ServiceMetadata) GetDynamicPricingMethodAssociated(methodFullName string) (pricingMethod string, isDynamicPricingEligible bool) {
	// Check if Method Level Options are defined for the given Service and method,
	// If Defined check if it's in the format supported, then return the full method Name
	// i.e., /package.service/method format; this will be directly fed in to the grpc called to made to
	// determine dynamic pricing
	if !config.GetBool(config.EnableDynamicPricing) {
		return
	}
	pricingMethod = metaData.DynamicPriceMethodMapping[methodFullName]
	if strings.Compare("", pricingMethod) == 0 {
		isDynamicPricingEligible = false
	} else {
		isDynamicPricingEligible = true
	}
	return
}

// IsModelTraining accept methodFullName, example "/example_service.Calculator/add"
func (metaData *ServiceMetadata) IsModelTraining(methodFullName string) (useModelTrainingEndPoint bool) {
	if !config.GetBool(config.ModelTrainingEnabled) {
		return false
	}
	return slices.Contains(metaData.TrainingMethods, methodFullName)
}

// getProtoDescriptors converts text of proto files to bufbuild linker
func getProtoDescriptors(protoFiles map[string]string) (linker.Files, error) {
	accessor := protocompile.SourceAccessorFromMap(protoFiles)
	r := protocompile.WithStandardImports(&protocompile.SourceResolver{Accessor: accessor})
	compiler := protocompile.Compiler{
		Resolver:       r,
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := compiler.Compile(context.Background(), slices.Collect(maps.Keys(protoFiles))...)
	if err != nil || fds == nil {
		zap.L().Error("failed to compile proto files"+errs.ErrDescURL(errs.InvalidProto), zap.Error(err))
		return nil, fmt.Errorf("failed to compile proto files: %v", err)
	}
	return fds, nil
}

func (metaData *ServiceMetadata) setServiceProto() (err error) {
	metaData.DynamicPriceMethodMapping = make(map[string]string, 0)
	metaData.TrainingMethods = make([]string, 0)
	var rawFile []byte

	// for backwards compatibility
	if metaData.ModelIpfsHash != "" {
		rawFile, err = ipfsutils.GetIpfsFile(metaData.ModelIpfsHash)
	}

	if metaData.ServiceApiSource != "" {
		rawFile, err = ipfsutils.ReadFile(metaData.ServiceApiSource)
	}

	if err != nil {
		zap.L().Error("Error in retrieving file from filecoin/ipfs", zap.Error(err))
		return err
	}

	metaData.ProtoFiles, err = ipfsutils.ReadProtoFilesCompressed(rawFile)
	if err != nil {
		return err
	}

	if metaData.ServiceType == "http" {

		if config.GetBool(config.ModelTrainingEnabled) {
			return errors.New("Training is not supported for HTTP services")
		}

		metaData.ProtoDescriptors, err = getProtoDescriptors(metaData.ProtoFiles)
		if err != nil {
			return err
		}
	}

	for _, file := range metaData.ProtoFiles {
		zap.L().Debug("Protofile", zap.String("file", file))

		// If Dynamic pricing is enabled, there will be mandatory checks on the service proto
		//this is to ensure that the standards on how one defines the methods to invoke is followed
		if config.GetBool(config.EnableDynamicPricing) || config.GetBool(config.ModelTrainingEnabled) {
			if srvProto, err := parseServiceProto(file); err != nil {
				return err
			} else {
				dynamicMethodMap, trainingMethodMap, err := buildDynamicPricingMethodsMap(srvProto)
				if err != nil {
					return err
				}
				metaData.DynamicPriceMethodMapping = dynamicMethodMap
				metaData.TrainingMethods = trainingMethodMap
			}
		}
	}

	return nil
}

// deprecated
func parseServiceProto(serviceProtoFile string) (*pproto.Proto, error) {
	reader := strings.NewReader(serviceProtoFile)
	parser := pproto.NewParser(reader)
	parsedProto, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return parsedProto, nil
}

// deprecated
func buildDynamicPricingMethodsMap(serviceProto *pproto.Proto) (dynamicPricingMethodMapping map[string]string,
	trainingMethodPricing []string, err error) {
	dynamicPricingMethodMapping = make(map[string]string, 0)
	trainingMethodPricing = make([]string, 0)
	var pkgName, serviceName, methodName string
	for _, elem := range serviceProto.Elements {
		//package is parsed earlier than service ( per documentation)
		if pkg, ok := elem.(*pproto.Package); ok {
			pkgName = pkg.Name
		}
		if service, ok := elem.(*pproto.Service); ok {
			serviceName = service.Name
			for _, serviceElements := range service.Elements {
				if rpcMethod, ok := serviceElements.(*pproto.RPC); ok {
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
