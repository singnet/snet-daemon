// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"io/ioutil"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

// calls the service heartbeat and relay the message to daemon
func callServiceHeartbeat(serviceURL string) (string, bool) {
	if isValidUrl(serviceURL) {
		response, err := http.Get(serviceURL)
		if err != nil {
			log.WithError(err).Fatal("The service request failed with an error.")
		} else {
			if response.StatusCode != http.StatusOK {
				log.Error("Wrong status code: %d", response.StatusCode)
				return "", false
			}
			log.Info("Service request processed successfully. ")
			serviceHeartbeat, _ := ioutil.ReadAll(response.Body)

			// TODO read and relay the heartbeat. it must be json string (which is URL Safe)
			if string(serviceHeartbeat) == "" {
				return string(serviceHeartbeat), false
			}
			return string(serviceHeartbeat), true
		}
	}
	//if invalid URL and returns Internal Server error
	return "{}", false
}

// Pushes the recoded metrics to Monitoring service
func callAndPostMetrics(serviceURL string, jsonMetrics string) bool {
	//TODO hit the metrics service URL and post the metrics data
	return false
}

// calls the correspanding the service to send the registration information
func callRegisterService(daemonID string, serviceURL string) (status bool) {
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

// isValidUrl tests a string to determine if it is a url or not.
func isValidUrl(urlToTest string) bool {
	_, err := url.ParseRequestURI(urlToTest)
	if err != nil {
		return false
	} else {
		return true
	}
}
