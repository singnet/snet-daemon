// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"bytes"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
)

// generates DaemonID nad returns i.e. DaemonID = HASH (Org Name, Service Name, daemon endpoint)
func getDaemonID() string {
	// TODO add the code to read from metadata and update Service Endpoint
	rawID := config.GetString(config.OrganizationName) + config.GetString(config.ServiceName) + config.GetString(config.DaemonEndPoint)

	// generate the keccak hash from given input string. Same as Ethereum hashes
	hash := crypto.Keccak256([]byte(rawID))

	// Convert hash byte array to hash string and return
	hashSize := bytes.IndexByte(hash, 0)
	return string(hash[:hashSize])
}

// New Daemon registration. Generates the DaemonID and use that as getting access token
func registerNewDaemon() (status bool) {
	daemonID := getDaemonID()

	// call the service and get the result
	status = callRegisterService(daemonID, config.GetString(config.MonitoringServiceEndpoint))

	// if registers successfully
	if status {
		log.Info("Daemon successfully registered with the monitoring service. ")
		return true
	}
	log.Info("Daemon unable to register with the monitoring service. ")
	// if unable to register, then throw an error.
	return false
}
