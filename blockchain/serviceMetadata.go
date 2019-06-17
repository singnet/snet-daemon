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
	PaymentExpirationThreshold *big.Int `json:"payment_expiration_threshold"`
	ModelIpfsHash              string   `json:"model_ipfs_hash"`
	MpeAddress                 string   `json:"mpe_address"`
	Pricing                    struct {
		PriceModel  string   `json:"price_model"`
		PackageName string `json:"package_name"`
		//Price in cogs has been retained only to support backward compatibility
		PriceInCogs *big.Int `json:"price_in_cogs"`
		Details    []struct {
			ServiceName   string `json:"service_name"`
			MethodPricing []struct {
				MethodName  string `json:"method_name"`
				PriceInCogs *big.Int    `json:"price_in_cogs"`
			} `json:"method_pricing"`
		} `json:"details"`
	} `json:"pricing"`
	Groups []struct {
		GroupName      string `json:"group_name"`
		GroupID        string `json:"group_id"`
		PaymentAddress string `json:"payment_address"`
	} `json:"groups"`
	Endpoints []struct {
		GroupName string `json:"group_name"`
		Endpoint  string `json:"endpoint"`
	} `json:"endpoints"`
	daemonReplicaGroupIDString string
	daemonReplicaGroupID       [32]byte
	daemonGroupName            string
	daemonEndPoint             string
	recipientPaymentAddress    common.Address
	multiPartyEscrowAddress    common.Address
}

func getRegistryAddressKey() common.Address {
	address := config.GetRegistryAddress()
	return common.HexToAddress(address)
}

func ServiceMetaData() *ServiceMetadata {
	var metadata *ServiceMetadata
	var err error
	if config.GetBool(config.BlockchainEnabledKey) {
		ipfsHash := string(getMetaDataUrifromRegistry())
		metadata, err = GetServiceMetaDataFromIPFS(FormatHash(ipfsHash))
	} else {
		//TO DO, have a snetd command to create a default metadata json file, for now just read from a local file
		// when block chain reading is disabled
		if metadata, err = ReadServiceMetaDataFromLocalFile("service_metadata.json");err != nil {
			fmt.Print("When Block chain is disabled it is mandatory to have a service_metadata.json file to start Daemon.Please refer to a sample file at https://github.com/singnet/snet-daemon/blob/master/service_metadata.json\n")
		}
	}
	if err != nil {
		log.WithError(err).
			Panic("error on determining service metadata from file")
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

func getMetaDataUrifromRegistry() []byte {
	ethClient, err := GetEthereumClient()
	defer ethClient.Close()
	registryContractAddress := getRegistryAddressKey()
	reg, err := NewRegistryCaller(registryContractAddress, ethClient.EthClient)
	if err != nil {
		log.WithError(err).WithField("registryContractAddress", registryContractAddress).
			Panic("Error instantiating Registry contract for the given Contract Address")
	}
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
	err = setDerivedFields(metaData)
	return metaData, err
}

func setDerivedFields(metaData *ServiceMetadata) error {
	err := setDaemonGroupName(metaData)
	if err != nil {
		return err
	}
	err = setDaemonGroupIDAndPaymentAddress(metaData)
	if err != nil {
		return err
	}
	setMultiPartyEscrowAddress(metaData)
	return nil

}

func setMultiPartyEscrowAddress(metaData *ServiceMetadata) {
	metaData.multiPartyEscrowAddress = common.HexToAddress(metaData.MpeAddress)

}


func setDaemonGroupName(metaData *ServiceMetadata) error {
	metaData.daemonGroupName = config.GetString(config.DaemonGroupName)
	//Make sure the group name specified is in the config matches to some group name in metadata
	for _, endpoints := range metaData.Endpoints {
		if strings.Compare(metaData.daemonGroupName, endpoints.GroupName) == 0 {
			return nil
		}
	}
	log.WithField("daemon group name does not match any of the Group Names in Metadata", metaData.daemonGroupName)
	return fmt.Errorf("please set the mandatory Daemon group Name corrrectly through the config %s ", config.DaemonGroupName)
}

func setDaemonGroupIDAndPaymentAddress(metaData *ServiceMetadata) error {
	groupName := metaData.GetDaemonGroupName()

	for _, group := range metaData.Groups {
		if strings.Compare(groupName, group.GroupName) == 0 {
			var err error
			metaData.daemonReplicaGroupIDString = group.GroupID
			metaData.daemonReplicaGroupID, err = ConvertBase64Encoding(group.GroupID)
			if err != nil {
				return err
			}
			metaData.recipientPaymentAddress = common.HexToAddress(group.PaymentAddress)
			return nil
		}
	}
	log.WithField("groupName", groupName)
	return fmt.Errorf("unable to determine the Daemon Group ID or the Recipient Payment Address, Daemon Group Name %s", groupName)

}

func (metaData *ServiceMetadata) GetDaemonEndPoint() string {
	return metaData.daemonEndPoint
}

func (metaData *ServiceMetadata) GetMpeAddress() common.Address {
	return metaData.multiPartyEscrowAddress
}

func (metaData *ServiceMetadata) GetPaymentExpirationThreshold() *big.Int {
	return metaData.PaymentExpirationThreshold
}


func (metaData *ServiceMetadata) GetDaemonGroupName() string {
	return metaData.daemonGroupName
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

func (metaData *ServiceMetadata) GetDaemonGroupID() [32]byte {
	return metaData.daemonReplicaGroupID
}

func (metaData *ServiceMetadata) GetDaemonGroupIDString() string {
	return metaData.daemonReplicaGroupIDString
}

func (metaData *ServiceMetadata) GetPaymentAddress() common.Address {
	return metaData.recipientPaymentAddress
}
