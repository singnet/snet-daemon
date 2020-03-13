// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// status enum
type Status int

// heartbeat URL Status
var isNoHeartbeatURL bool

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
	DaemonVersion    string `json:"daemonVersion"`
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

// set the no heartbeat URL State
func SetNoHeartbeatURLState(state bool) {
	isNoHeartbeatURL = state
}

// validates the heartbeat configurations
func ValidateHeartbeatConfig() error {
	//initialize the url state to false
	SetNoHeartbeatURLState(false)
	// check if the configured type is not supported
	hbType := config.GetString(config.ServiceHeartbeatType)
	if hbType != "grpc" && hbType != "http" && hbType != "https" && hbType != "none" && hbType != "" {
		return fmt.Errorf("unrecognized heartbet service type : '%+v'", hbType)
	}
	// if the URLs are empty, or hbtype is None or empty consider it as not configured
	if hbType == "" || hbType == "none" || config.GetString(config.HeartbeatServiceEndpoint) == "" {
		SetNoHeartbeatURLState(true)
	} else if !config.IsValidUrl(config.GetString(config.HeartbeatServiceEndpoint)) {
		return errors.New("service endpoint must be a valid URL")
	}
	return nil
}

// prepares the heartbeat, which includes calling to underlying service DAemon is serving
func GetHeartbeat(serviceURL string, serviceType string, serviceID string) (heartbeat DaemonHeartbeat, err error) {
	heartbeat = DaemonHeartbeat{GetDaemonID(), strconv.FormatInt(getEpochTime(), 10), Online.String(), "{}", config.GetVersionTag()}
	var curResp = `{"serviceID":"` + serviceID + `","status":"NOT_SERVING"}`
	if serviceType == "none" || serviceType == "" || isNoHeartbeatURL {
		curResp = `{"serviceID":"` + serviceID + `","status":"SERVING"}`
	} else {
		var svcHeartbeat []byte
		if serviceType == "grpc" {
			var response grpc_health_v1.HealthCheckResponse_ServingStatus
			response, err = callgRPCServiceHeartbeat(serviceURL)
			//Standardize this as well on the response being sent
			heartbeat.Status = response.String()
		} else if serviceType == "http" || serviceType == "https" {
			svcHeartbeat, err = callHTTPServiceHeartbeat(serviceURL)
		}
		if err != nil {
			heartbeat.Status = Warning.String()
			// send the alert if service heartbeat fails
			notification := &Notification{
				Recipient: config.GetString(config.AlertsEMail),
				Details:   err.Error(),
				Timestamp: time.Now().String(),
				Message:   "Problem in calling Service Heartbeat endpoint.",
				Component: "Daemon",
				DaemonID:  GetDaemonID(),
				Level:     "ERROR",
			}
			notification.Send()
		} else {
			log.Debugf("Service %s status : %s", serviceURL, svcHeartbeat)
			curResp = string(svcHeartbeat)
		}
	}
	heartbeat.ServiceHeartbeat = curResp
	return heartbeat, err
}

// Heartbeat request handler function : upon request it will hit the service for status and
// wraps the results in daemons heartbeat
func HeartbeatHandler(rw http.ResponseWriter, r *http.Request) {
	// read the heartbeat service type and corresponding URL
	serviceType := config.GetString(config.ServiceHeartbeatType)
	serviceURL := config.GetString(config.HeartbeatServiceEndpoint)
	serviceID := config.GetString(config.ServiceId)
	heartbeat, _ := GetHeartbeat(serviceURL, serviceType, serviceID)
	err := json.NewEncoder(rw).Encode(heartbeat)
	if err != nil {
		log.WithError(err).Infof("Failed to write heartbeat message.")
	}
}

// Check implements `service Health`.
func (service *DaemonHeartbeat) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {

	heartbeat, err := GetHeartbeat(config.GetString(config.HeartbeatServiceEndpoint), config.GetString(config.ServiceHeartbeatType),
		config.GetString(config.ServiceId))

	if strings.Compare(heartbeat.Status, Online.String()) == 0 {
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN}, errors.New("Service heartbeat unknown " + err.Error())
}

// Watch implements `service Watch todo for later`.
func (service *DaemonHeartbeat) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	return nil
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
