// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"github.com/singnet/snet-daemon/metrics/services"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_callgRPCServiceHeartbeat(t *testing.T) {
	serviceURL := "localhost:35000"
	heartbeat, err := callgRPCServiceHeartbeat(serviceURL)
	assert.False(t, err != nil)

	assert.NotEqual(t, `{}`, string(heartbeat), "Service Heartbeat must not be empty.")
	assert.Equal(t, `{"serviceID":"sample1","status":"SERVING"}`, string(heartbeat),
		"Unexpected service heartbeat")

	var sHeartbeat grpc_health_v1.HeartbeatMsg
	err = json.Unmarshal(heartbeat, &sHeartbeat)
	assert.True(t, err != nil)
	assert.Equal(t, "sample1", sHeartbeat.ServiceID, "Unexpected service ID")

	serviceURL = "localhost:26000"
	heartbeat, err = callgRPCServiceHeartbeat(serviceURL)
	assert.True(t, err != nil)
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
	serviceURL := "https://demo3208027.mockable.io/register"
	daemonID := GetDaemonID()

	result := callRegisterService(daemonID, serviceURL)
	assert.Equal(t, true, result)

	serviceURL = "https://demo3208027.mockable.io/registererror"
	result = callRegisterService(daemonID, serviceURL)
	assert.Equal(t, false, result)

}
