package ipfsutils

import (
	"testing"

	"github.com/ipfs/kubo/client/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	_ "github.com/singnet/snet-daemon/v6/config"
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

func (suite *IpfsUtilsTestSuite) TestGetProtoFiles() {
	hash := "ipfs://Qmc32Gi3e62gcw3fFfRPidxrGR7DncNki2ptfh9rVESsTc"
	data, err := ReadFile(hash)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), data)
	protoFiles, err := ReadProtoFilesCompressed(data)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), protoFiles)
}

func TestIsGzipFile(t *testing.T) {
	// Valid gzip header
	gzipData := []byte{0x1F, 0x8B, 0x08, 0x00}
	if !isGzipFile(gzipData) {
		t.Errorf("Expected true for valid gzip header")
	}

	// Invalid gzip header
	invalidData := []byte{0x00, 0x01, 0x02, 0x03}
	if isGzipFile(invalidData) {
		t.Errorf("Expected false for invalid gzip header")
	}
}

func (suite *IpfsUtilsTestSuite) TestReadFiles() {
	// For testing purposes, a hash is used from the calculator service.
	hash := "QmeyrQkEyba8dd4rc3jrLd5pEwsxHutfH2RvsSaeSMqTtQ"
	data, err := GetIpfsFile(hash)
	assert.NotNil(suite.T(), data)
	assert.Nil(suite.T(), err)

	protoFiles, err := ReadProtoFilesCompressed(data)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), protoFiles)

	expectedProtoFiles := map[string]string{"example_service.proto": `syntax = "proto3";

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

	assert.EqualValues(suite.T(), expectedProtoFiles, protoFiles)
}
