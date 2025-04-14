package main

import (
	"github.com/ethereum/go-ethereum/accounts/abi/abigen"
	contracts "github.com/singnet/snet-ecosystem-contracts"
	"log"
	"os"
)

// Generate smart-contracts golang bindings
func main() {
	bindContent, err := abigen.Bind(
		[]string{"MultiPartyEscrow", "Registry", "FetchToken"},
		[]string{
			string(contracts.GetABIClean(contracts.MultiPartyEscrow)),
			string(contracts.GetABIClean(contracts.Registry)),
			string(contracts.GetABIClean(contracts.FetchToken))},
		//string(contracts.GetABIClean(contracts.SingularityNetToken))},
		[]string{
			string(contracts.GetBytecodeClean(contracts.MultiPartyEscrow)),
			string(contracts.GetBytecodeClean(contracts.Registry)),
			string(contracts.GetBytecodeClean(contracts.FetchToken))},
		//string(contracts.GetBytecodeClean(contracts.SingularityNetToken))},
		nil, "blockchain", nil, nil)
	if err != nil {
		log.Fatalf("Failed to generate binding: %v", err)
	}

	if err := os.WriteFile("snet-contracts.go", []byte(bindContent), 0600); err != nil {
		log.Fatalf("Failed to write ABI binding: %v", err)
	}
}
