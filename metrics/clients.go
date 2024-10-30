// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"bytes"
	"context"
	"errors"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"net/http"
	"time"

	"github.com/singnet/snet-daemon/v5/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Response struct {
	ServiceID string `json:"serviceID"`
	Status    string `json:"status"`
}

// Calls a gRPC endpoint for heartbeat (gRPC Client)
func callgRPCServiceHeartbeat(serviceUrl string) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	// Set up a connection to the server.
	conn, err := grpc.NewClient(serviceUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		zap.L().Warn("unable to connect to grpc endpoint", zap.Error(err))
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING, err
	}
	defer conn.Close()
	// create the client instance
	client := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := grpc_health_v1.HealthCheckRequest{Service: config.GetString(config.ServiceId)}
	resp, err := client.Check(ctx, &req)
	if err != nil {
		zap.L().Warn("error in calling the heartbeat service", zap.Error(err))
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	return resp.Status, nil
}

// calls the service heartbeat and relay the message to daemon (HTTP client for heartbeat)
func callHTTPServiceHeartbeat(serviceURL string) ([]byte, error) {
	response, err := http.Get(serviceURL)
	if err != nil {
		zap.L().Info("the service request failed with an error", zap.Error(err))
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		zap.L().Warn("wrong status code", zap.Int("StatusCode", response.StatusCode))
		return nil, errors.New("unexpected error with the service")
	}
	// Read the response
	serviceHeartbeat, _ := io.ReadAll(response.Body)
	//Check if we got empty response
	if string(serviceHeartbeat) == "" {
		return nil, errors.New("empty service response")
	}
	zap.L().Info("response received", zap.Any("response", serviceHeartbeat))
	return serviceHeartbeat, nil
}

// calls the corresponding the service to send the registration information
func callRegisterService(daemonID string, serviceURL string) (status bool) {
	//Send the Daemon ID and the Network ID to register the Daemon
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(buildPayLoadForServiceRegistration()))
	if err != nil {
		zap.L().Info("unable to create register service request", zap.Error(err))
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Access-Token", daemonID)

	// sending the post request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		zap.L().Info("unable to reach registration service", zap.Error(err))
		return false
	}
	// process the response and set the Authorization token
	daemonAuthorizationToken, status = getTokenFromResponse(response)
	zap.L().Debug("daemon authorization token", zap.Any("value", daemonAuthorizationToken))
	return
}

func buildPayLoadForServiceRegistration() []byte {
	payload := &RegisterDaemonPayload{DaemonID: GetDaemonID()}
	body, _ := ConvertStructToJSON(payload)
	zap.L().Debug("build payload for service registration", zap.Binary("body", body))
	return body
}
