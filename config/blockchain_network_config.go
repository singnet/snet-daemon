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
	DefaultBlockChainNetworkConfig string = `
{
  "local":{
    "ethereum_json_rpc_endpoint":"http://localhost:8545",
    "network_id":"999999",
    "registry_address_key":"0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2"
  },
  "kovan":{
    "ethereum_json_rpc_endpoint":"https://kovan.infura.io",
    "network_id":"42"
  },
  "ropsten":{
    "ethereum_json_rpc_endpoint":"https://ropsten.infura.io",
    "network_id":"3"
  },
  "rinkeby":{
    "ethereum_json_rpc_endpoint":"https://rinkeby.infura.io",
    "network_id":"4"
  },
  "main":{
    "ethereum_json_rpc_endpoint":"https://mainnet.infura.io",
    "network_id":"1"
  }
}`
	BlockChainNetworkFileName  = "blockchain_network_config.json"
	EthereumJsonRpcEndpointKey = "ethereum_json_rpc_endpoint"
	NetworkId                  = "network_id"
	RegistryAddressKey         = "registry_address_key"
	//Read this from a config file path todo
	RegistryJsonFileName = "resources/blockchain/node_modules/singularitynet-platform-contracts/networks/Registry.json"
)

var networkSelected *NetworkSelected

func determineNetworkSelected(data []byte) {
	dynamicBinding := map[string]interface{}{}
	networkName := GetString(BlockChainNetworkSelected)
	err := json.Unmarshal(data, &dynamicBinding)
	if err != nil {
		panic(fmt.Sprintf("cannot parse the JSON configuation file %v for the network %v  to determine the network selected: %v",
			BlockChainNetworkFileName, GetString(BlockChainNetworkSelected), err))
	}
	networkSelected = &NetworkSelected{
		//Get the Network Name selected in config ( snetd.config.json) , Based on this retrieve the Registry address ,
		//Ethereum End point and Network ID mapped
		NetworkName:             networkName,
		RegistryAddressKey:      getRegistryAddressfromJSON(dynamicBinding[networkName].(map[string]interface{})[RegistryAddressKey]),
		EthereumJSONRPCEndpoint: fmt.Sprintf("%v", dynamicBinding[networkName].(map[string]interface{})[EthereumJsonRpcEndpointKey]),
		NetworkId:               fmt.Sprintf("%v", dynamicBinding[networkName].(map[string]interface{})[NetworkId]),
	}
}

//Check if the Registry address was set in the block chain network config, then we use this address as the contract address
//the address is usually set for local testing , for other network types like ropsten or kovan or main , the system will automatically
//figure out the contract address
func getRegistryAddressfromJSON(address interface{}) string {
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
func setRegistryAddress() {
	//if address is already set in the config file and has been initialized , then skip the setting process
	if len(networkSelected.RegistryAddressKey) > 0 {
		return
	}
	data, err := ReadFromFile(RegistryJsonFileName)
	if err != nil {
		panic(fmt.Sprintf("cannot find the file at %v for the network %v configuation file to read the address config: %v",
			RegistryJsonFileName, GetString(BlockChainNetworkSelected), err))
	}
	networkSelected.RegistryAddressKey = getRegistryAddressFromJson(data)
}

func getRegistryAddressFromJson(data []byte) string {
	m := map[string]interface{}{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		panic(fmt.Sprintf("cannot parse the JSON file at %v for the %v configuation file to read the address config: %v",
			RegistryJsonFileName, GetString(BlockChainNetworkSelected), err))
	}
	return fmt.Sprintf("%v", m[GetNetworkId()].(map[string]interface{})["address"])
}

//Read the file given file, if the file is not found  ,then return back an error
func ReadFromFile(filename string) ([]byte, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file: %v", filename)
	}
	return file, nil

}

//Read from the block chain network config json
func setAndValidateBlockChainNetworkDetails() {
	data, err := ReadFromFile(BlockChainNetworkFileName)
	if err != nil {
		data = []byte(DefaultBlockChainNetworkConfig)
	}
	determineNetworkSelected(data)
	setRegistryAddress()
	log.Infof("blockchain_network_selected: %v BlockChainNetwork Registry Address:%v  Ethereum end point:%v ",
		GetString(BlockChainNetworkSelected), GetRegistryAddress(), GetBlockChainEndPoint())
}
