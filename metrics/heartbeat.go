// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// status enum
type Status int
const (
	Offline Status = 0 // Returns if none of the services are online
	Online Status = 1 // Returns if any of the services is online
	Warnings Status = 2 // if daemon has issues in extracting the service state
	Critical Status = 3 // if the daemon main thread killed or any other critical issues
)

// define heartbeat data model. Service Status JSON object Array marshalled to a string
type HeartbeatMessage struct {
	DaemonID  		string	`json:"daemonID"`
	Timestamp   	string	`json:"timestamp"`
	Status      	string	`json:"status"`
	ServiceStatus	string	`json:"serviceStatus"`
}

// Converts the enum index into enum names
func (state Status) String() string {
	// declare an array of strings. operator counts how many items in the array (4)
	listStatus := [...]string{ "Offline", "Online", "Warnings", "Critical",}

	// → `state`: It's one of the values of Status constants.
	// prevent panicking in case of `status` is out of range of Status
	if state < Offline || state > Critical {
		return "Unknown"
	}
	// return the status string constant from the array above.
	return listStatus[state]
}

// returns the epoch UTC timestamp from the current system time
func getEpochTime() int64 {
	return time.Now().UTC().Unix()
}


// preares the heartbeat, which includes calling to underlying service DAemon is serving
func getHeartbeat() (HeartbeatMessage, bool) {
	hearbeat := HeartbeatMessage{getDaemonID(), strconv.FormatInt(getEpochTime(),10), Online.String(),"[{}]"}
	//TODO Read the service metadata and get the service URL
	serviceURL := "https://reqres.in/api/users/2"
	svcHeartbeat, isSuccess := getServiceHeartbeat(serviceURL)
	if isSuccess {
		//TODO convert the service call response to sevice status array
		log.Info("Service %s status : %s", serviceURL, svcHeartbeat)
	} else {
		// TODO maintain the previous state. if not avialble then relay status : unknown

		hearbeat.Status = Warnings.String()
	}
	return hearbeat, true
}

// Heartbeat request handler function : upon request it will hit the service for status and
// wraps the results in daemons heartbeat
func heartbeatHandler(rw http.ResponseWriter, r *http.Request) {
	heartbeat, status := getHeartbeat()
	if !status {
		log.Warningf("Unable to get Heartbeat. ")
	}
	err := json.NewEncoder(rw).Encode(heartbeat)
	if err != nil {
		log.Fatalf("Failed to write heartbeat message. Reason: %s", err.Error())
	}
}
