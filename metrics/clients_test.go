// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"github.com/singnet/snet-daemon/metrics/services"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_callgRPCServiceHeartbeat(t *testing.T) {
	type args struct {
		grpcAddress string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := callgRPCServiceHeartbeat(tt.args.grpcAddress)
			if (err != nil) != tt.wantErr {
				t.Errorf("callgRPCServiceHeartbeat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("callgRPCServiceHeartbeat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_callHTTPServiceHeartbeat(t *testing.T) {
	serviceURL := "http://demo3208027.mockable.io/heartbeat"

	heartbeat, err := callHTTPServiceHeartbeat(serviceURL)
	assert.False(t, err != nil)

	assert.NotEqual(t, string(heartbeat), `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(t, string(heartbeat), `{"serviceID":"SERVICE001", "status":"SERVING"}`,
		"Unexpected service heartbeat")

	var sHeartbeat grpc_health_v1.HeartbeatMsg
	err = json.Unmarshal(heartbeat, &sHeartbeat)
	assert.True(t, err != nil)
	assert.Equal(t, sHeartbeat.ServiceID, "SERVICE001", "Unexpected service ID")

	serviceURL = "http://demo3208027.mockable.io"

	heartbeat, err = callHTTPServiceHeartbeat(serviceURL)
	assert.True(t, err != nil)
}

func Test_callRegisterService(t *testing.T) {
	type args struct {
		daemonID   string
		serviceURL string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotStatus := callRegisterService(tt.args.daemonID, tt.args.serviceURL); gotStatus != tt.wantStatus {
				t.Errorf("callRegisterService() = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}
