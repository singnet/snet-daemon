// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"net/url"
)

// generates DaemonID nad returns i.e. DaemonID = HASH (Org Name, Service Name, daemon endpoint)
func getDaemonID() (string) {
	// TODO add the code to read from metadata and generate HASH
	return ""
}

// New Daemon registration. Generates the DaemonID and use that as getting access token
func registerNewDaemon() (status bool) {
	daemonID := getDaemonID()

	// call the service and get the result
	status = callRegisterService(daemonID, config.GetString(config.MonitoringServiceEndpoint))

	// if registers successfully
	if  (status) {
		log.Info("Daemon successfully registered with the monitoring service. ")
		//TODO add the validate Daemon ID to Config and use it for the session
		return true
	}
	log.Info("Daemon unable to register with the monitoring service. ")
	// if unable to register, then throw an error.
	return  false
}

// calls the correspanding the service to send the registration information
func callRegisterService (daemonID string, serviceURL string) (status bool) {
	// Validate the URL
	_, err := url.ParseRequestURI(serviceURL)
	if err != nil {
		log.WithError(err).Fatal("Unable to register Daemon. Invalid service URL.")
	}

	//TODO Send this Daemon to monitoring service and wait for confirmation.
	//	   If service returns True then registration is successful, else its a failure
	log.Info("Registration service called with result  ")
	return false
}