package blockchain

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/ipfsutils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/big"
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
            "group_id": "EoFmN3nvaXpf6ew8jJbIPVghE5NXfYupFF7PkRmVyGQ="
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

	freeCallSignerAddress common.Address
	isfreeCallAllowed     bool
	freeCallsAllowed      int
}

type OrganizationGroup struct {
	Endpoints      []string  `json:"endpoints"`
	GroupID        string    `json:"group_id"`
	GroupName      string    `json:"group_name"`
	Pricing        []Pricing `json:"pricing"`
	FreeCalls      int       `json:"free_calls"`
	FreeCallSigner string    `json:"free_call_signer_address"`
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
	file, err := ioutil.ReadFile(filename)
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
		if !common.IsHexAddress((metaData.defaultGroup.FreeCallSigner)) {
			return fmt.Errorf("MetaData does not have 'free_call_signer_address defined correctly")
		}
		if !config.IsValidUrl(config.GetString(config.FreeCallEndPoint)) {
			return fmt.Errorf("Please specify a valid end point for 'free_call_end_point' tracking usage of free calls ")
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
