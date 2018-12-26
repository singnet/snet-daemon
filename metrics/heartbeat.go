// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

//go:generate protoc -I services/ services/heartbeat.proto --go_out=plugins=grpc:services

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

// status enum
type Status int

const (
	Offline  Status = 0 // Returns if none of the services are online
	Online   Status = 1 // Returns if any of the services is online
	Warning  Status = 2 // if daemon has issues in extracting the service state
	Critical Status = 3 // if the daemon main thread killed or any other critical issues
)

// define heartbeat data model. Service Status JSON object Array marshalled to a string
type DaemonHeartbeat struct {
	DaemonID         string `json:"daemonID"`
	Timestamp        string `json:"timestamp"`
	Status           string `json:"status"`
	ServiceHeartbeat string `json:"serviceheartbeat"`
}

// Converts the enum index into enum names
func (state Status) String() string {
	// declare an array of strings. operator counts how many items in the array (4)
	listStatus := [...]string{"Offline", "Online", "Warning", "Critical"}

	// → `state`: It's one of the values of Status constants.
	// prevent panicking in case of `status` is out of range of Status
	if state < Offline || state > Critical {
		return "Unknown"
	}
	// return the status string constant from the array above.
	return listStatus[state]
}

// prepares the heartbeat, which includes calling to underlying service DAemon is serving
func GetHeartbeat(serviceURL string, serviceType string, serviceID string) DaemonHeartbeat {
	heartbeat := DaemonHeartbeat{GetDaemonID(), strconv.FormatInt(getEpochTime(), 10), Online.String(), "{}"}

	var curResp = `{"serviceID":"` + serviceID + `","status":"NOT_SERVING"}`
	var svcHeartbeat []byte
	var err error
	// if daemon type is grpc, then call grpc heartbeat, else go for HTTP service heartbeat
	if serviceType == "grpc" {
		svcHeartbeat, err = callgRPCServiceHeartbeat(serviceURL)
	} else {
		svcHeartbeat, err = callHTTPServiceHeartbeat(serviceURL)
	}
	if err != nil {
		heartbeat.Status = Warning.String()
		// send the alert if service heartbeat fails
		notification := &Notification{
			Recipient: config.GetString(config.AlertsEMail),
			Details:   err.Error(),
			Timestamp: time.Now().String(),
			Message:   "Problem in calling Service Heatbeat endpoint.",
			Component: "Daemon",
			DaemonID:  GetDaemonID(),
			Level:     "ERROR",
		}
		notification.Send()
	} else {
		log.Infof("Service %s status : %s", serviceURL, svcHeartbeat)
		curResp = string(svcHeartbeat)
	}
	heartbeat.ServiceHeartbeat = curResp
	return heartbeat
}

// Heartbeat request handler function : upon request it will hit the service for status and
// wraps the results in daemons heartbeat
func HeartbeatHandler(rw http.ResponseWriter, r *http.Request) {
	// read the heartbeat service type and corresponding URL
	serviceType := config.GetString(config.ServiceHeartbeatType)
	serviceURL := config.GetString(config.HeartbeatServiceEndpoint)
	serviceID := config.ServiceId
	heartbeat := GetHeartbeat(serviceURL, serviceType, serviceID)
	err := json.NewEncoder(rw).Encode(heartbeat)
	if err != nil {
		log.WithError(err).Infof("Failed to write heartbeat message.")
	}
}

/*
service heartbeat/grpc heartbeat
{"serviceID":"sample1", "status":"SERVING"}

daemon heartbeat
{
  "daemonID": "3a4ebeb75eace1857a9133c7a50bdbb841b35de60f78bc43eafe0d204e523dfe",
  "timestamp": "2018-12-26 22:50:13.4569654 +0000 UTC",
  "status": "Online",
  "serviceheartbeat": "{\"serviceID\":\"sample1\", \"status\":\"SERVING\"}"
}
*/
