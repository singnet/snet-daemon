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

const IpfsPrefix = "ipfs://"

type ServiceMetadata struct {
	Version                    int      `json:"version"`
	DisplayName                string   `json:"display_name"`
	Encoding                   string   `json:"encoding"`
	ServiceType                string   `json:"service_type"`
    Groups                     []OrganizationGroup `json:"groups"`
	ModelIpfsHash              string   `json:"model_ipfs_hash"`
	MpeAddress                 string   `json:"mpe_address"`

	multiPartyEscrowAddress    common.Address
	defaultPricing Pricing

	defaultGroup                OrganizationGroup
}

type OrganizationGroup struct {
	Endpoints []string `json:"endpoints"`
	GroupID   string   `json:"group_id"`
	GroupName      string  `json:"group_name"`
	Pricing   []Pricing  `json:"pricing"`
}
type Pricing struct {
	PriceModel  string `json:"price_model"`
	PriceInCogs *big.Int    `json:"price_in_cogs,omitempty"`
	PackageName string `json:"package_name,omitempty"`
	Default     bool   `json:"default,omitempty"`
	PricingDetails []PricingDetails `json:"details,omitempty"`
}

type PricingDetails struct {
	ServiceName   string               `json:"service_name"`
	MethodPricing []MethodPricing `json:"method_pricing"`
}
type MethodPricing struct {
	MethodName  string `json:"method_name"`
	PriceInCogs *big.Int    `json:"price_in_cogs"`
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
		metadata = &ServiceMetadata{Encoding:"proto",ServiceType:"grpc"}
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
		return nil,err
	}

	return metaData, err
}

func setDerivedFields(metaData *ServiceMetadata) (err error) {
	if err= setDefaultPricing(metaData); err != nil {
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
			return  nil
		}
	}
	err = fmt.Errorf("group name %v in config is invalid, "+
		"there was no group found with this name in the metadata", groupName)
	log.WithError(err)
	return  err
}

func setDefaultPricing(metaData *ServiceMetadata) (err error) {
	if err = setGroup(metaData);err != nil {
		return err
	}
	for _, pricing := range metaData.defaultGroup.Pricing {
		if pricing.Default {
			metaData.defaultPricing = pricing
			return  nil
		}
	}
	err = fmt.Errorf("MetaData does not have the default pricing set ")
	log.WithError(err)
	return err
}

func setMultiPartyEscrowAddress(metaData *ServiceMetadata) {
	metaData.multiPartyEscrowAddress = common.HexToAddress(metaData.MpeAddress)
	//set the checksum address ( standardized way)
	metaData.MpeAddress = metaData.multiPartyEscrowAddress.Hex()

}


func (metaData *ServiceMetadata) GetMpeAddress() common.Address {
	return metaData.multiPartyEscrowAddress
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

