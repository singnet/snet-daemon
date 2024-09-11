package config

import (
	"encoding/json"
	"fmt"
	"os"

	contracts "github.com/singnet/snet-ecosystem-contracts"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type NetworkSelected struct {
	NetworkName             string
	EthereumJSONRPCEndpoint string
	NetworkId               string
	RegistryAddressKey      string
}

const (
	BlockChainNetworkFileName  = "resources/blockchain_network_config.json"
	EthereumJsonRpcEndpointKey = "ethereum_json_rpc_endpoint"
	NetworkId                  = "network_id"
	RegistryAddressKey         = "registry_address_key"
)

var networkSelected = &NetworkSelected{}
var networkIdNameMapping string

func determineNetworkSelected(data []byte) (err error) {
	dynamicBinding := map[string]any{}
	networkName := GetString(BlockChainNetworkSelected)
	if err = json.Unmarshal(data, &dynamicBinding); err != nil {
		return err
	}
	//Get the Network Name selected in config ( snetd.config.json) , Based on this retrieve the Registry address ,
	//Ethereum End point and Network ID mapped to
	networkSelected.NetworkName = networkName
	networkSelected.RegistryAddressKey = getDetailsFromJsonOrConfig(dynamicBinding[networkName].(map[string]any)[RegistryAddressKey], RegistryAddressKey)
	networkSelected.EthereumJSONRPCEndpoint = getDetailsFromJsonOrConfig(dynamicBinding[networkName].(map[string]any)[EthereumJsonRpcEndpointKey], EthereumJsonRpcEndpointKey)
	networkSelected.NetworkId = fmt.Sprintf("%v", dynamicBinding[networkName].(map[string]any)[NetworkId])

	return err
}

// Check if the value set in the  config file, if yes, then we use it as is
// else we derive the value from the JSON parsed
func getDetailsFromJsonOrConfig(details any, configName string) string {
	if len(GetString(configName)) > 0 {
		return GetString(configName)
	}
	if details == nil {
		return ""
	}
	return fmt.Sprintf("%v", details)
}

// Get the Network ID associated  with the network selected
func GetNetworkId() string {
	return networkSelected.NetworkId
}

// Get the block chain end point associated with the Network selected
func GetBlockChainEndPoint() string {
	return networkSelected.EthereumJSONRPCEndpoint
}

// Get the Registry address of the contract
func GetRegistryAddress() string {
	return networkSelected.RegistryAddressKey
}

// Read the Registry address from JSON file passed
func setRegistryAddress() (err error) {
	//if the address is already set in the config file and has been initialized, then skip the setting process
	if len(networkSelected.RegistryAddressKey) > 0 {
		return
	}

	data := contracts.GetNetworksClean(contracts.Registry)
	//This value will be set when the binary across multiple platforms is built

	//this is only for your local set up and testing
	//This is only for local set up testing
	//if data, err = ReadFromFile(fileName); err != nil {
	//	return fmt.Errorf("cannot parse the JSON file at %v to determine registry address of %v network, the error is :%v",
	//		fileName, GetString(BlockChainNetworkSelected), err)
	//}

	if err = deriveDatafromJSON(data); err != nil {
		return err
	}
	return nil
}

func deriveDatafromJSON(data []byte) (err error) {
	m := map[string]any{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return fmt.Errorf("cannot parse the registry JSON file for the network %v , the error is : %v",
			GetString(BlockChainNetworkSelected), err)
	}

	if m[GetNetworkId()] == nil {
		return errors.New("cannot find registry address from JSON for the selected network")
	}

	networkSelected.RegistryAddressKey = fmt.Sprintf("%v", m[GetNetworkId()].(map[string]any)["address"])

	zap.L().Info("Derive data from JSON", zap.String("Network", GetString(BlockChainNetworkSelected)),
		zap.String("NetwrokId", GetNetworkId()),
		zap.String("RegistryAddress", GetRegistryAddress()),
		zap.String("Blockchain endpoint", GetBlockChainEndPoint()))
	return nil
}

// Read the file given file, if the file is not found  ,then return back an error
func ReadFromFile(filename string) ([]byte, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		if file, err = os.ReadFile("../" + filename); err != nil {
			return nil, errors.Wrapf(err, "could not read file: %v", filename)
		}
	}
	return file, nil
}

// Read from the blockchain network config json
func setBlockChainNetworkDetails(fileName string) (err error) {
	var data []byte
	if len(networkIdNameMapping) > 0 {
		data = []byte(networkIdNameMapping)
	} else {
		data, err = ReadFromFile(fileName)
		if err != nil {
			return fmt.Errorf("cannot read the file %v , error is %v",
				fileName, err)
		}
	}
	if err = determineNetworkSelected(data); err == nil {
		err = setRegistryAddress()
	}
	return err
}
