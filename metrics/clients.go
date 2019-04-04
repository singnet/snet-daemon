// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"bytes"
	"context"
	"errors"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"io/ioutil"
	"net/http"
	"time"
)

type Response struct {
	ServiceID string `json:"serviceID"`
	Status    string `json:"status"`
}

// Calls a gRPC endpoint for heartbeat (gRPC Client)
func callgRPCServiceHeartbeat(serviceUrl string) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(serviceUrl, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("unable to connect to grpc endpoint: %v", err)
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING, err
	}
	defer conn.Close()
	// create the client instance
	client := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := grpc_health_v1.HealthCheckRequest{Service:config.GetString(config.ServiceId)}
	resp, err := client.Check(ctx,&req)
	if err != nil {
		log.WithError(err).Warningf("error in calling the heartbeat service : %v", err)
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	return resp.Status,nil
}

// calls the service heartbeat and relay the message to daemon (HTTP client for heartbeat)
func callHTTPServiceHeartbeat(serviceURL string) ([]byte, error) {
	response, err := http.Get(serviceURL)
	if err != nil {
		log.WithError(err).Info("the service request failed with an error: %v", err)
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		log.Warningf("wrong status code: %d", response.StatusCode)
		return nil, errors.New("unexpected error with the service")
	}
	// Read the response
	serviceHeartbeat, _ := ioutil.ReadAll(response.Body)
	//Check if we got empty response
	if string(serviceHeartbeat) == "" {
		return nil, errors.New("empty service response")
	}
	log.Infof("response received : %v", serviceHeartbeat)
	return serviceHeartbeat, nil
}

// calls the corresponding the service to send the registration information
func callRegisterService(daemonID string, serviceURL string) (status bool) {
	//Send the Daemon ID and the Network ID to register the Daemon
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(buildPayLoadForServiceRegistration()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Access-Token", daemonID)
	if err != nil {
		log.WithError(err).Infof("unable to create register service request : %v", err)
		return false
	}
	// sending the post request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.WithError(err).Info("unable to reach registration service : %v", err)
		return false
	}
	// process the response and set the Authorization token
	daemonAuthorizationToken, status = getTokenFromResponse(response)
	log.Debugf("daemonAuthorizationToken %v", daemonAuthorizationToken)
	return
}

func buildPayLoadForServiceRegistration() []byte {
	payload := &RegisterDaemonPayload{DaemonID: GetDaemonID()}
	body, _ := ConvertStructToJSON(payload)
	log.Debugf("buildPayLoadForServiceRegistration() %v", string(body))
	return body
}
