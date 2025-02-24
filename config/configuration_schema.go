package config

import (
	"encoding/json"
	"github.com/spf13/viper"
	"strings"
)

//TO DO, Work in Progress; this defines the complete Schema of the Daemon Configuration
//Defined the schema of a few configurations to give an example; you need to define the schema of a given configuration here to see it on the UI

// Used to map the attribute values to a struct
type ConfigurationDetails struct {
	Name          string //the key of the attribute becomes the Value of the Name
	Mandatory     bool   `json:"mandatory"`
	DefaultValue  string `json:"value"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Editable      bool   `json:"editable"`
	RestartDaemon bool   `json:"restart_daemon"`
	Section       string `json:"section"`
}

const DefaultDaemonConfigurationSchema = `
{
  "registry_address_key": {
    "mandatory": true,
    "value": "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2",
    "description": "Ethereum address of the Registry contract instance.This is auto determined if not specified based on the blockchain_network_selected If a value is specified ,it will be used and no attempt will be made to auto determine the registry address",
    "type": "address",
    "editable": false,
    "restart_daemon": true,
    "section": "blockchain"
  }
,
  "ethereum_json_rpc_http_endpoint": {
    "mandatory": true,
    "value": "https://sepolia.infura.io/v3",
    "description": "Endpoint to which daemon sends ethereum JSON-RPC requests; Based on the network selected blockchain_network_selected the end point is auto determined.",
    "type": "url",
    "editable": true,
    "restart_daemon": true,
    "section": "blockchain"
  },

  "blockchain_network_selected": {
    "mandatory": true,
    "value": "local",
    "description": "Name of the network to be used for Daemon possible values are one of (sepolia, goerli, main or local). Daemon will automatically read the Registry address associated with this network For local network ( you can also specify the registry address manually),see the blockchain_network_config.json",
    "type": "string",
    "editable": true,
    "restart_daemon": true,
    "section": "blockchain"
  },

  "service_endpoint": {
    "mandatory": true,
    "value": "http://127.0.0.1:7003",
    "description": "endpoint to which requests should be proxied for handling by service.",
    "type": "url",
    "editable": true,
    "restart_daemon": false,
    "section": "general"
  }
}`

func isLeafNodeKey(key string) (bool, string) {
	if strings.Contains(key, ".restart_daemon") {
		newKey := strings.Replace(key, ".restart_daemon", "", -1)
		return true, newKey
	}
	return false, ""
}

func GetConfigurationSchema() ([]ConfigurationDetails, error) {
	allConfigurations := make([]ConfigurationDetails, 0) //CHECK THIS
	defaultConfigSchema := viper.New()
	if err := ReadConfigFromJsonString(defaultConfigSchema, DefaultDaemonConfigurationSchema); err != nil {
		return nil, err
	}
	for _, key := range defaultConfigSchema.AllKeys() {
		//Find out if the given key is the key of a Leaf or not.
		if isLeaf, leafKey := isLeafNodeKey(key); isLeaf {
			configurationDetailsJSON, _ := ConvertStructToJSON(defaultConfigSchema.Get(leafKey))
			configDetails := &ConfigurationDetails{}
			configDetails.Name = leafKey
			_ = json.Unmarshal(configurationDetailsJSON, configDetails)
			allConfigurations = append(allConfigurations, *configDetails)
		}
	}
	return allConfigurations, nil
}

// ConvertStructToJSON converts the passed datastructure to a JSON
func ConvertStructToJSON(payLoad any) ([]byte, error) {
	b, err := json.Marshal(&payLoad)
	if err != nil {
		return nil, err
	}
	return b, nil
}
