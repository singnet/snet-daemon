// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/semyon-dev/cmux"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/stretchr/testify/assert"
)

type HeartBeatTestSuite struct {
	suite.Suite
	serviceURL   string
	server       *grpc.Server
	viper        *viper.Viper
	currentBlock func() (*big.Int, error)
	trainingMD   func() (*training.TrainingMetadata, error)
}

func (suite *HeartBeatTestSuite) TearDownSuite() {
	suite.server.GracefulStop()
}

func (suite *HeartBeatTestSuite) SetupSuite() {

	suite.viper = config.Vip()
	suite.viper.Set(config.ServiceId, "YOUR_SERVICE_ID")
	suite.viper.Set(config.OrganizationId, "YOUR_ORG_ID")
	suite.viper.Set(config.DaemonGroupName, "default_group")

	suite.serviceURL = "http://localhost:1111"
	suite.currentBlock = blockchain.NewMockProcessor(true).CurrentBlock
	suite.server = setAndServe()
	suite.trainingMD = func() (*training.TrainingMetadata, error) {
		return &training.TrainingMetadata{TrainingInProto: true, TrainingEnabled: true}, nil
	}
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
	config.Vip().Set(config.HeartbeatServiceEndpoint, suite.serviceURL)
	// Creating a request to pass to the handler.  the third parameter is nil since we are not passing any parameters to service
	request, err := http.NewRequest("GET", suite.serviceURL+"/heartbeat", nil)
	if err != nil {
		assert.Fail(suite.T(), "Unable to create request payload for testing the Heartbeat Handler")
	}

	// Creating a ResponseRecorder to record the response.
	response := httptest.NewRecorder()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		HeartbeatHandler(writer, func() (*training.TrainingMetadata, error) {
			return suite.trainingMD()
		}, nil, suite.currentBlock)
	})

	// Since it is a basic http handler, we can call ServeHTTP method directly and pass request and response.
	handler.ServeHTTP(response, request)

	// test the responses
	assert.Equal(suite.T(), http.StatusOK, response.Code, "handler returned wrong status code")
	heartbeat, err := io.ReadAll(response.Body)
	assert.NoError(suite.T(), err)

	var dHeartbeat DaemonHeartbeat
	err = json.Unmarshal(heartbeat, &dHeartbeat)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), dHeartbeat.TrainingMetadataData.TrainingInProto)
	assert.NotNil(suite.T(), dHeartbeat, "heartbeat must not be nil")

	assert.Equal(suite.T(), Offline.String(), dHeartbeat.Status, "Invalid State")
	//assert.NotEqual(suite.T(), Offline.String(), dHeartbeat.Status, "Invalid State")

	assert.Equal(suite.T(), "20e986a77adb1ab0900dce6f554128496aa26be1e212d682f37898bae226fcfc", dHeartbeat.DaemonID,
		"Incorrect daemon ID")

	assert.NotEqual(suite.T(), dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), `{"serviceID":"YOUR_SERVICE_ID","status":"NOT_SERVING"}`, dHeartbeat.ServiceHeartbeat, "Unexpected service heartbeat")
}

func (suite *HeartBeatTestSuite) Test_GetHeartbeat() {
	serviceURL := suite.serviceURL + "/heartbeat"
	serviceType := "http"
	serviveID := "SERVICE001"

	dHeartbeat, _ := GetHeartbeat(serviceURL, serviceURL, serviceType, serviveID, suite.trainingMD, nil, suite.currentBlock)
	assert.NotNil(suite.T(), dHeartbeat, "heartbeat must not be nil")

	assert.Equal(suite.T(), dHeartbeat.Status, Online.String(), "Invalid State")
	assert.NotEqual(suite.T(), dHeartbeat.Status, Offline.String(), "Invalid State")

	assert.Equal(suite.T(), "20e986a77adb1ab0900dce6f554128496aa26be1e212d682f37898bae226fcfc", dHeartbeat.DaemonID,
		"Incorrect daemon ID")

	assert.NotEqual(suite.T(), dHeartbeat.ServiceHeartbeat, `{}`, "Service Heartbeat must not be empty.")
	assert.Equal(suite.T(), `{"serviceID":"SERVICE001","status":"SERVING"}`, dHeartbeat.ServiceHeartbeat,
		"Unexpected service heartbeat")

	var sHeartbeat DaemonHeartbeat
	err := json.Unmarshal([]byte(dHeartbeat.ServiceHeartbeat), &sHeartbeat)
	assert.True(suite.T(), err == nil)
	assert.Equal(suite.T(), sHeartbeat.Status, grpc_health_v1.HealthCheckResponse_SERVING.String())

	// check with some timeout URL
	serviceURL = "http://localhost:1234"
	dHeartbeat2, _ := GetHeartbeat(serviceURL, serviceURL, serviceType, serviveID, suite.trainingMD, nil, suite.currentBlock)
	assert.NotNil(suite.T(), dHeartbeat2, "heartbeat must not be nil")

	assert.Equal(suite.T(), Offline.String(), dHeartbeat2.Status, "Invalid State")
	assert.NotEqual(suite.T(), Online.String(), dHeartbeat2.Status, "Invalid State")
	assert.NotNil(suite.T(), dHeartbeat2.TrainingMetadataData)

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

func (suite *HeartBeatTestSuite) TestValidateHeartbeatConfig() {
	err := ValidateHeartbeatConfig("", suite.serviceURL+"/heartbeat")
	assert.NoError(suite.T(), err)

	err = ValidateHeartbeatConfig("http", "h://invalid_url")
	assert.Error(suite.T(), err)
}
