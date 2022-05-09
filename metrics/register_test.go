// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"testing"

	"github.com/stretchr/testify/assert"
)

type RegisterTestSuite struct {
	suite.Suite
	serviceURL string
	server     *grpc.Server
}

func (suite *RegisterTestSuite) TearDownSuite() {
	suite.server.GracefulStop()
}
func (suite *RegisterTestSuite) SetupSuite() {
	SetNoHeartbeatURLState(false)
	suite.serviceURL = "http://localhost:1111"
	suite.server = setAndServe()
}

func TestRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterTestSuite))
}
func TestGetDaemonID(t *testing.T) {
	daemonID := GetDaemonID()

	assert.NotNil(t, daemonID, "daemon ID must not be nil")
	assert.NotEmpty(t, daemonID, "daemon ID must not be empty")
	assert.Equal(t, "091a31bb4b808c574fb5e158923f3067c4e23e805252699fb5946121e7ca1506", daemonID)
	assert.NotEqual(t, "48d343313a1e06093c81830103b45496cc7c277f321049e9ee632fd03207d042", daemonID)
}

func (suite *RegisterTestSuite) TestRegisterDaemon() {
	serviceURL := "http://localhost:1111/register"

	result := RegisterDaemon(serviceURL)
	assert.Equal(suite.T(), true, result)

	serviceURL = "https://localhost:9999/registererror"
	result = RegisterDaemon(serviceURL)
	assert.Equal(suite.T(), false, result)
}

func (suite *RegisterTestSuite) TestSetDaemonGrpId() {
	grpid := "group01"
	SetDaemonGrpId(grpid)
	assert.NotNil(suite.T(), daemonGroupId)
	assert.NotEmpty(suite.T(), daemonGroupId)
	assert.Equal(suite.T(), "group01", daemonGroupId)
	assert.NotEqual(suite.T(), "some wrong group id", daemonGroupId)
}
