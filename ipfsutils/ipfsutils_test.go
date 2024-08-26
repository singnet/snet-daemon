package ipfsutils

import (
	"testing"

	"github.com/ipfs/kubo/client/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	_ "github.com/singnet/snet-daemon/config"
)

type IpfsUtilsTestSuite struct {
	suite.Suite
	ipfsClient *rpc.HttpApi
}

func TestIpfsUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(IpfsUtilsTestSuite))
}

func (suite *IpfsUtilsTestSuite) BeforeTest() {
	suite.ipfsClient = GetIPFSClient()
	assert.NotNil(suite.T(), suite.ipfsClient)
}

func (suite *IpfsUtilsTestSuite) TestReadFiles() {
	// For testing purposes, a hash is used from the calculator service.
	hash := "QmeyrQkEyba8dd4rc3jrLd5pEwsxHutfH2RvsSaeSMqTtQ"
	data := GetIpfsFile(hash)
	assert.NotNil(suite.T(), data)

	protoFiles, err := ReadFilesCompressed(data)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), protoFiles)

	excpectedProtoFiles := []string{`syntax = "proto3";

package example_service;

message Numbers {
    float a = 1;
    float b = 2;
}

message Result {
    float value = 1;
}

service Calculator {
    rpc add(Numbers) returns (Result) {}
    rpc sub(Numbers) returns (Result) {}
    rpc mul(Numbers) returns (Result) {}
    rpc div(Numbers) returns (Result) {}
}`}

	assert.Equal(suite.T(), excpectedProtoFiles, protoFiles)
}
