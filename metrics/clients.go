// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	pb "github.com/singnet/snet-daemon/metrics/services"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Response struct {
	ServiceName string `json:"serviceName"`
	Status      string `json:"status"`
}

// Calls a gRPC endpoint for heartbeat (gRPC Client)
func callgRPCServiceHeartbeat(grpcAddress string) ([]byte, error) {

	// Set up a connection to the server.
	conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("Unable to connect to grpc endpoint: %v", err)
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
		log.WithError(err).Warningf("Error in calling the heartbeat service : %v", err)
	}
	//convert enum to string, because json marshal doesnt do it
	responseConv := &Response{ServiceName: resp.ServiceName, Status: resp.Status.String()}
	jsonResp, err := json.Marshal(responseConv)
	if err != nil {
		log.WithError(err).Warningf("Invalid service response : %v", err)
	}
	log.Infof("Service heartbeat received : %s", string(jsonResp))
	return jsonResp, nil
}

// calls the service heartbeat and relay the message to daemon (HTTP client for heartbeat)
func callHTTPServiceHeartbeat(serviceURL string) ([]byte, error) {
	response, err := http.Get(serviceURL)
	if err != nil {
		log.WithError(err).Info("The service request failed with an error.")
	} else {
		if response.StatusCode != http.StatusOK {
			log.WithError(err).Warningf("Wrong status code: %d", response.StatusCode)
			return []byte(""), errors.New("Unexpected error with the service.")
		}
		log.Infof("Service request processed successfully. ")
		serviceHeartbeat, _ := ioutil.ReadAll(response.Body)

		if string(serviceHeartbeat) == "" {
			return serviceHeartbeat, errors.New("Invalid service response")
		}
		return serviceHeartbeat, nil
	}
	return []byte(""), errors.New("Invalid service response")
}

// calls the correspanding the service to send the registration information
func callRegisterService(daemonID string, serviceURL string) (status bool) {
	// prepare the request payload
	input := []byte(`{"daemonID":"` + daemonID + `"}`)
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(input))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.WithError(err).Infof("Unable to create register service request")
	}
	// sending the post request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.WithError(err).Info("unable to reach metrics service")
	} else {
		if response.StatusCode != http.StatusOK {
			log.WithError(err).Warningf("Wrong status code: %d", response.StatusCode)
		}
		log.Infof("Service request processed successfully. ")

		// read the response body
		body, _ := ioutil.ReadAll(response.Body)
		result, _ := strconv.ParseBool(string(body))
		return result
	}
	defer response.Body.Close()
	return false
}

// sends a notification to the user via notification service
func callNotificationService(jsonAlert []byte, serviceURL string) bool {
	//prepare the request payload
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(jsonAlert))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.WithError(err).Warningf("Unable to create notification service request")
	}
	// sending the post request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warningf("unable to reach notification service")
	} else {
		if response.StatusCode != http.StatusOK {
			log.WithError(err).Warningf("Wrong status code: %d", response.StatusCode)
		}
		log.Infof("Service request processed successfully. ")

		// read the response body
		body, _ := ioutil.ReadAll(response.Body)
		result, _ := strconv.ParseBool(string(body))
		return result
	}
	defer response.Body.Close()
	return false
}

// Pushes the recoded metrics to Monitoring service
func callAndPostMetrics(serviceURL string, jsonMetrics string) bool {
	//TODO hit the metrics service URL and post the metrics data
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
