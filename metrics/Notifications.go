// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
)

// define heartbeat data model. Service Status JSON object Array marshalled to a string
type Notification struct {
	DaemonID  string `json:"component_id"`
	Timestamp string `json:"timestamp"`
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	Details   string `json:"details"`
	Component string `json:"component"`
	Type      string `json:"type"`
	Level     string `json:"level"`
}

// function for sending an alert to a given endpoint
func (alert *Notification) Send() {
	serviceURL := config.GetString(config.NotificationServiceEndpoint)
	// convert the notification struct to json
	jsonAlert, err := ConvertStructToJSON(alert)
	log.Infof("Notification : %v", string(jsonAlert))
	if err != nil {
		log.WithError(err).Warningf("Json conversion error : %v", err)
	} else {
		// based on the notification success/failure
		status := Publish(jsonAlert, serviceURL)
		if !status {
			log.Infof("Unable to send notification. ")
		}
	}
}

/*
service request
{
    "recipient":"raam.comm@gmail.com",
    "message":"From the API",
    "details":"From the API",
    "component":"daemon",
    "component_id":"ad",
    "type":"INFO",
    "level":"10"
}

service response
true/false
*/
