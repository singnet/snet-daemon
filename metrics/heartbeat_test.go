// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"google.golang.org/grpc/health/grpc_health_v1"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_String(t *testing.T) {
	assert.Equal(t, Online.String(), "Online", "Invalid enum string conversion")
	assert.NotEqual(t, Online.String(), "Offline", "Invalid enum string conversion")
}

func TestHeartbeatHandler(t *testing.T) {
	// Creating a request to pass to the handler.  third parameter is nil since we are not passing any parameters to service
	request, err := http.NewRequest("GET", "/heartbeat", nil)
	if err != nil {
		assert.Fail(t, "Unable to create request payload for testing the Heartbeat Handler")
	}

	// Creating a ResponseRecorder to record the response.
	response := httptest.NewRecorder()
	handler := http.HandlerFunc(HeartbeatHandler)

	// Since it is basic http handler, we can call ServeHTTP method directly and pass request and response.
	handler.ServeHTTP(response, request)

	//test the responses
	assert.Equal(t, http.StatusOK, response.Code, "handler returned wrong status code")
	heartbeat, _ := ioutil.ReadAll(response.Body)

	var dHeartbeat DaemonHeartbeat
	err = json.Unmarshal([]byte(heartbeat), &dHeartbeat)
	assert.False(t, err != nil)
	assert.NotNil(t, dHeartbeat, "heartbeat must not be nil")

	assert.Equal(t, dHeartbeat.Status, Online.String(), "Invalid State")
	assert.NotEqual(t, dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(t, dHeartbeat.DaemonID, "2188ffe79222a44083c315dbb6bc82f3292fa76131b226a85c8ed11361a2406f",
		"Incorrect daemon ID")

	assert.NotEqual(t, dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(t, dHeartbeat.ServiceHeartbeat, `{"serviceID":"SERVICE001", "status":"SERVING"}`,
		"Unexpected service heartbeat")
}

func Test_GetHeartbeat(t *testing.T) {
	serviceURL := "http://demo3208027.mockable.io/heartbeat"
	serviceType := "http"
	serviveID := "SERVICE001"

	dHeartbeat,_ := GetHeartbeat(serviceURL, serviceType, serviveID)
	assert.NotNil(t, dHeartbeat, "heartbeat must not be nil")

	assert.Equal(t, dHeartbeat.Status, Online.String(), "Invalid State")
	assert.NotEqual(t, dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(t, dHeartbeat.DaemonID, "2188ffe79222a44083c315dbb6bc82f3292fa76131b226a85c8ed11361a2406f",
		"Incorrect daemon ID")

	assert.NotEqual(t, dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(t, dHeartbeat.ServiceHeartbeat, `{"serviceID":"SERVICE001", "status":"SERVING"}`,
		"Unexpected service heartbeat")

	var sHeartbeat DaemonHeartbeat
	err := json.Unmarshal([]byte(dHeartbeat.ServiceHeartbeat), &sHeartbeat)
	assert.True(t, err == nil)
	assert.Equal(t, sHeartbeat.Status, grpc_health_v1.HealthCheckResponse_SERVING.String())

	// check with some timeout URL
	serviceURL = "http://demo3208027.mockable.io"
	dHeartbeat,_ = GetHeartbeat(serviceURL, serviceType, serviveID)
	assert.NotNil(t, dHeartbeat, "heartbeat must not be nil")

	assert.Equal(t, dHeartbeat.Status, Warning.String(), "Invalid State")
	assert.NotEqual(t, dHeartbeat.Status, Online.String(), "Invalid State")

	assert.NotEqual(t, dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(t, dHeartbeat.ServiceHeartbeat, `{"serviceID":"SERVICE001","status":"NOT_SERVING"}`,
		"Unexpected service heartbeat")
}

func validateHeartbeat(t *testing.T, dHeartbeat DaemonHeartbeat) {
	assert.NotNil(t, dHeartbeat, "heartbeat must not be nil")

	assert.Equal(t, dHeartbeat.Status, Online.String(), "Invalid State")
	assert.NotEqual(t, dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(t, dHeartbeat.DaemonID, "cc48d343313a1e06093c81830103b45496749e9ee632fd03207d042c277f3210",
		"Incorrect daemon ID")

	assert.NotEqual(t, dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(t, dHeartbeat.ServiceHeartbeat, `{"serviceID":"SERVICE001", "status":"SERVING"}`,
		"Unexpected service heartbeat")
}

func TestSetNoHeartbeatURLState(t *testing.T) {
	SetNoHeartbeatURLState(true)
	assert.Equal(t, true, isNoHeartbeatURL)

	SetNoHeartbeatURLState(false)
	assert.Equal(t, false, isNoHeartbeatURL)
}

func TestValidateHeartbeatConfig(t *testing.T) {
	err := ValidateHeartbeatConfig()
	assert.Nil(t, err)
	assert.Equal(t, false, isNoHeartbeatURL)
}
