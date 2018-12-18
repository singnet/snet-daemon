// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
)

// define heartbeat data model. Service Status JSON object Array marshalled to a string
type Notification struct {
	DaemonID  string `json:"daemonID"`
	Timestamp string `json:"timestamp"`
	To        string `json:"to"`
	Message   string `json:"message"`
}

// function for sending an alert to a given endpoint
func (alert *Notification) Send() bool {
	serviceURL := config.GetString(config.NotificationServiceEndpoint) + "/register"
	status := false

	// convert the notification struct to json
	jsonAlert, err := json.Marshal(alert)
	log.Info(string(jsonAlert))
	if err != nil {
		log.WithError(err).Warningf("Json conversion error.")
	} else {
		//check whether given address is valid or not
		if !isValidUrl(serviceURL) {
			log.Warningf("Invalid service URL %s", serviceURL)
		} else {
			// based on the notification success/failure
			status := callNotificationService(jsonAlert, serviceURL)
			if status {
				log.Infof("Notification sent. ")
				return status
			}
			log.Infof("Unable to send notification. ")
		}
	}
	return status
}

/*
service request
{"daemonID":"3a4ebeb75eace1857a9133c7a50bdbb841b35de60f78bc43eafe0d204e523dfe","timestamp":"1544913544","to":"rdr1207@gmail.com","message":"Unexpected Error in Daemon metrics"}

service response
true/false
*/
