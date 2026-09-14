package ipfsutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ipfs/kubo/client/rpc"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	zap.ReplaceGlobals(zap.New(zapcore.NewNopCore()))
}

func withTestConfig(t *testing.T, overrides map[string]string) {
	t.Helper()
	originals := make(map[string]string)
	for k := range overrides {
		originals[k] = config.GetString(k)
	}
	for k, v := range overrides {
		config.Vip().Set(k, v)
	}
	t.Cleanup(func() {
		for k, v := range originals {
			config.Vip().Set(k, v)
		}
	})
}

func startMockServer(handler http.HandlerFunc) *httptest.Server {
	ts := httptest.NewServer(handler)
	return ts
}

func TestGetIpfsFile_InvalidHash(t *testing.T) {
	withTestConfig(t, map[string]string{config.IpfsEndpoint: "http://127.0.0.1:1"})

	data, err := GetIpfsFile("not-a-valid-cid")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetIpfsFile_SendError(t *testing.T) {
	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: "http://127.0.0.1:1",
		config.IpfsTimeout:  "1",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetIpfsFile_ServerError(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("command not found"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestGetIpfsFile_ServerJSONError(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"Message":"internal error","Code":1}`))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Nil(t, data)
	assert.Error(t, err)
}

func TestGetIpfsFile_Success(t *testing.T) {
	content := []byte("IPFS file content")
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(content)
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestGetIpfsFile_ReadBodyError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, bufrw, err := hijacker.Hijack()
		if err != nil {
			return
		}
		fmt.Fprint(bufrw, "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
		fmt.Fprint(bufrw, "6\r\nhello!\r\n")
		bufrw.Flush()
		conn.Close()
	}))
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetIPFSClient_Success(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Version":"0.1.0"}`))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	client := GetIPFSClient()
	assert.NotNil(t, client)
}

// ========== compressed.go ==========

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
