// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"context"
	"encoding/json"
	"errors"
	pb "github.com/singnet/snet-daemon/metrics/services"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
	"time"
)

type Response struct {
	ServiceID string `json:"serviceID"`
	Status    string `json:"status"`
}

// Calls a gRPC endpoint for heartbeat (gRPC Client)
func callgRPCServiceHeartbeat(grpcAddress string) ([]byte, error) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("unable to connect to grpc endpoint: %v", err)
		return nil, err
	}
	defer conn.Close()

	// create the client instance
	client := pb.NewHeartbeatClient(conn)
	// connect to the server and call the required method
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//call the heartbeat rpc method
	resp, err := client.Check(ctx, &pb.Empty{})
	if err != nil {
		log.WithError(err).Warningf("error in calling the heartbeat service : %v", err)
		return nil, err
	}
	//convert enum to string, because json marshal doesnt do it
	responseConv := &Response{ServiceID: resp.ServiceID, Status: resp.Status.String()}
	jsonResp, err := json.Marshal(responseConv)
	if err != nil {
		log.Infof("response received : %v", responseConv)
		log.WithError(err).Warningf("invalid service response : %v", err)
		return nil, err
	}
	log.Infof("service heartbeat received : %s", string(jsonResp))
	return jsonResp, nil
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
	req, err := http.NewRequest("POST", serviceURL, nil)
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
	// process the response
	return checkForSuccessfulResponse(response)
}
