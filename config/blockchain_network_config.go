package config

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

type NetworkSelected struct {
	NetworkName             string
	EthereumJSONRPCEndpoint string
	NetworkId               string
	RegistryAddressKey      string
}

const (
	BlockChainNetworkFileName  = "blockchain_network_config.json"
	EthereumJsonRpcEndpointKey = "ethereum_json_rpc_endpoint"
	NetworkId                  = "network_id"
	RegistryAddressKey         = "registry_address_key"
	RegistryAddressFileName    = "Registry.json"
)

var networkSelected *NetworkSelected

func determineNetworkSelected(data []byte) (err error) {
	dynamicBinding := map[string]interface{}{}
	networkName := GetString(BlockChainNetworkSelected)
	if err = json.Unmarshal(data, &dynamicBinding); err != nil {
		return err
	}

	networkSelected = &NetworkSelected{
		//Get the Network Name selected in config ( snetd.config.json) , Based on this retrieve the Registry address ,
		//Ethereum End point and Network ID mapped to
		NetworkName:             networkName,
		RegistryAddressKey:      getRegistryAddress(dynamicBinding[networkName].(map[string]interface{})[RegistryAddressKey]),
		EthereumJSONRPCEndpoint: geEthereumJSONRPCEndpoint(dynamicBinding[networkName].(map[string]interface{})[EthereumJsonRpcEndpointKey]),
		NetworkId:               fmt.Sprintf("%v", dynamicBinding[networkName].(map[string]interface{})[NetworkId]),
	}
	return err
}


func geEthereumJSONRPCEndpoint(endpoint interface{}) string {
	if (len(GetString(EthereumJsonRpcEndpointKey))> 0) {
		return GetString(EthereumJsonRpcEndpointKey)
	}
	if endpoint == nil {
		return ""
	}
	return fmt.Sprintf("%v", endpoint)
}

//Check if the Registry address was set in the  config, then we use this address as the contract address
//the address is usually set for local testing , for other network types like ropsten or kovan or main , the system will automatically
//figure out the contract address unless explicitly overridden in the config file
func getRegistryAddress(address interface{}) string {

	if (len(GetString(RegistryAddressKey))> 0) {
		return GetString(RegistryAddressKey)
	}
	if address == nil {
		return ""
	}
	return fmt.Sprintf("%v", address)
}

//Get the Network ID associated  with the network selected
func GetNetworkId() (string) {
	return networkSelected.NetworkId
}

//Get the block chain end point associated with the Network selected
func GetBlockChainEndPoint() string {
	return networkSelected.EthereumJSONRPCEndpoint
}

//Get the Registry address of the contract
func GetRegistryAddress() string {
	return networkSelected.RegistryAddressKey
}

//Read the Registry address from JSON file ( file will be under networks folder)
func setRegistryAddress() (err error) {
	var data []byte

	//if address is already set in the config file and has been initialized , then skip the setting process
	if len(networkSelected.RegistryAddressKey) > 0 {
		return
	}

	if data, err = ReadFromFile(RegistryAddressFileName); err != nil {
		return fmt.Errorf("cannot parse the JSON file at %v to determine registry address of %v network, the error is :%v",
			RegistryAddressFileName, GetString(BlockChainNetworkSelected), err)
	}

	m := map[string]interface{}{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return fmt.Errorf("cannot parse the JSON file at %v for the %v configuation file to read the address config: %v",
			RegistryAddressFileName, GetString(BlockChainNetworkSelected), err)
	}

	networkSelected.RegistryAddressKey = fmt.Sprintf("%v", m[GetNetworkId()].(map[string]interface{})["address"])

	log.Infof("The Network specified is :%v, and maps to the Network Id %v, the determined Registry address is %v and block chain end point is %v",
		GetString(BlockChainNetworkSelected), GetNetworkId(), GetRegistryAddress(), GetBlockChainEndPoint())
	return nil
}

//Read the file given file, if the file is not found  ,then return back an error
func ReadFromFile(filename string) ([]byte, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		if file, err = ReadFromFile("../" + BlockChainNetworkFileName); err != nil {
			return nil, errors.Wrapf(err, "could not read file: %v", filename)
		}
	}
	return file, nil

}

//Read from the block chain network config json
func setBlockChainNetworkDetails() (err error) {
	data, err := ReadFromFile(BlockChainNetworkFileName)
	if err != nil {
		return fmt.Errorf("cannot read the file %v , error is %v",
			BlockChainNetworkFileName, err)
	}
	if err = determineNetworkSelected(data); err == nil {
		err = setRegistryAddress()
	}
	return err
}
