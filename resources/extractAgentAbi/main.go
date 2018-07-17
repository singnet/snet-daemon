package main

import (
	"encoding/json"
	"io/ioutil"
)

func main() {
	if rawJson, err := ioutil.ReadFile("resources/blockchain/node_modules/singularitynet-alpha-blockchain/Agent.json"); err == nil {
		parsedJson := make(map[string]interface{})
		json.Unmarshal(rawJson, &parsedJson)
		abi := parsedJson["abi"]
		if abiBytes, err := json.Marshal(abi); err == nil {
			ioutil.WriteFile("resources/Agent.abi", abiBytes, 0644)
		}
	}
}
