// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"fmt"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/soheilhy/cmux"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type HeartBeatTestSuite struct {
	suite.Suite
	serviceURL string
	server     *grpc.Server
}

func (suite *HeartBeatTestSuite) TearDownSuite() {
	suite.server.GracefulStop()
}
func (suite *HeartBeatTestSuite) SetupSuite() {
	SetNoHeartbeatURLState(false)
	suite.serviceURL = "http://localhost:1111"
	suite.server = setAndServe()
}
func setAndServe() (server *grpc.Server) {
	server = grpc.NewServer()
	ch := make(chan int)
	go func() {
		lis, err := net.Listen("tcp", ":1111")
		if err != nil {
			panic(err)
		}
		mux := cmux.New(lis)
		grpcWebServer := grpcweb.WrapServer(server, grpcweb.WithCorsForRegisteredEndpointsOnly(false))
		httpHandler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsAcceptableGrpcCorsRequest(req) {
				grpcWebServer.ServeHTTP(resp, req)
			} else {
				if strings.Split(req.URL.Path, "/")[1] == "register" {
					resp.Header().Set("Access-Control-Allow-Origin", "*")
					fmt.Fprintln(resp, "Registering service...... ")
				} else if strings.Split(req.URL.Path, "/")[1] == "heartbeat" {
					resp.Header().Set("Access-Control-Allow-Origin", "*")
					fmt.Fprint(resp, "{\"serviceID\":\"SERVICE001\",\"status\":\"SERVING\"}")
				} else {
					http.NotFound(resp, req)

				}
			}
		})
		daemonHeartBeat := &DaemonHeartbeat{DaemonID: "metrics.GetDaemonID()", DaemonVersion: "test version"}
		grpc_health_v1.RegisterHealthServer(server, daemonHeartBeat)

		httpL := mux.Match(cmux.HTTP1Fast())
		grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
		go server.Serve(grpcL)
		go http.Serve(httpL, httpHandler)
		go mux.Serve()
		ch <- 0
	}()

	_ = <-ch
	return
}

func TestHeartBeatTestSuite(t *testing.T) {
	suite.Run(t, new(HeartBeatTestSuite))
}

func (suite *HeartBeatTestSuite) TestStatus_String() {
	assert.Equal(suite.T(), Online.String(), "Online", "Invalid enum string conversion")
	assert.NotEqual(suite.T(), Online.String(), "Offline", "Invalid enum string conversion")
}

func (suite *HeartBeatTestSuite) TestHeartbeatHandler() {
	SetNoHeartbeatURLState(false)
	// Creating a request to pass to the handler.  third parameter is nil since we are not passing any parameters to service
	request, err := http.NewRequest("GET", "/heartbeat", nil)
	if err != nil {
		assert.Fail(suite.T(), "Unable to create request payload for testing the Heartbeat Handler")
	}

	// Creating a ResponseRecorder to record the response.
	response := httptest.NewRecorder()
	handler := http.HandlerFunc(HeartbeatHandler)

	// Since it is basic http handler, we can call ServeHTTP method directly and pass request and response.
	handler.ServeHTTP(response, request)

	//test the responses
	assert.Equal(suite.T(), http.StatusOK, response.Code, "handler returned wrong status code")
	heartbeat, _ := ioutil.ReadAll(response.Body)

	var dHeartbeat DaemonHeartbeat
	err = json.Unmarshal([]byte(heartbeat), &dHeartbeat)
	assert.False(suite.T(), err != nil)
	assert.NotNil(suite.T(), dHeartbeat, "heartbeat must not be nil")

	assert.Equal(suite.T(), dHeartbeat.Status, Warning.String(), "Invalid State")
	assert.NotEqual(suite.T(), dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(suite.T(), dHeartbeat.DaemonID, "f940de0eb33eeddb283ac725478900deac24151b019e496c476d59f72c38abb3",
		"Incorrect daemon ID")

	assert.NotEqual(suite.T(), dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), dHeartbeat.ServiceHeartbeat, `{"serviceID":"ExampleServiceId","status":"NOT_SERVING"}`,
		"Unexpected service heartbeat")
}

func (suite *HeartBeatTestSuite) Test_GetHeartbeat() {
	serviceURL := suite.serviceURL + "/heartbeat"
	serviceType := "http"
	serviveID := "SERVICE001"

	dHeartbeat, _ := GetHeartbeat(serviceURL, serviceType, serviveID)
	assert.NotNil(suite.T(), dHeartbeat, "heartbeat must not be nil")

	assert.Equal(suite.T(), dHeartbeat.Status, Online.String(), "Invalid State")
	assert.NotEqual(suite.T(), dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(suite.T(), dHeartbeat.DaemonID, "f940de0eb33eeddb283ac725478900deac24151b019e496c476d59f72c38abb3",
		"Incorrect daemon ID")

	assert.NotEqual(suite.T(), dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), dHeartbeat.ServiceHeartbeat, `{"serviceID":"SERVICE001","status":"SERVING"}`,
		"Unexpected service heartbeat")

	var sHeartbeat DaemonHeartbeat
	err := json.Unmarshal([]byte(dHeartbeat.ServiceHeartbeat), &sHeartbeat)
	assert.True(suite.T(), err == nil)
	assert.Equal(suite.T(), sHeartbeat.Status, grpc_health_v1.HealthCheckResponse_SERVING.String())

	// check with some timeout URL
	serviceURL = "http://localhost:1234"
	SetNoHeartbeatURLState(false)
	dHeartbeat2, _ := GetHeartbeat(serviceURL, serviceType, serviveID)
	assert.NotNil(suite.T(), dHeartbeat2, "heartbeat must not be nil")

	assert.Equal(suite.T(), dHeartbeat2.Status, Warning.String(), "Invalid State")
	assert.NotEqual(suite.T(), dHeartbeat2.Status, Online.String(), "Invalid State")

	assert.NotEqual(suite.T(), dHeartbeat2.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), dHeartbeat2.ServiceHeartbeat, `{"serviceID":"SERVICE001","status":"NOT_SERVING"}`,
		"Unexpected service heartbeat")
}

func (suite *HeartBeatTestSuite) validateHeartbeat(dHeartbeat DaemonHeartbeat) {
	assert.NotNil(suite.T(), dHeartbeat, "heartbeat must not be nil")

	assert.Equal(suite.T(), dHeartbeat.Status, Online.String(), "Invalid State")
	assert.NotEqual(suite.T(), dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(suite.T(), dHeartbeat.DaemonID, "cc48d343313a1e06093c81830103b45496749e9ee632fd03207d042c277f3210",
		"Incorrect daemon ID")

	assert.NotEqual(suite.T(), dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), dHeartbeat.ServiceHeartbeat, `{"serviceID":"SERVICE001", "status":"SERVING"}`,
		"Unexpected service heartbeat")
}

func (suite *HeartBeatTestSuite) TestSetNoHeartbeatURLState() {
	SetNoHeartbeatURLState(true)
	assert.Equal(suite.T(), true, isNoHeartbeatURL)

	SetNoHeartbeatURLState(false)
	assert.Equal(suite.T(), false, isNoHeartbeatURL)
}

func (suite *HeartBeatTestSuite) TestValidateHeartbeatConfig() {
	err := ValidateHeartbeatConfig()
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), true, isNoHeartbeatURL)
}
