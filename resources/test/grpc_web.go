package test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/singnet/snet-daemon/resources/test/echoservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGRPCWeb(t *testing.T, h harness, port string) {
	job := h.createSignedJob()
	time.Sleep(time.Second)

	req := &echoservice.EchoEnvelope{Message: "from gRPC-Web"}
	payload, err := proto.Marshal(req)
	assert.NoError(t, err, "Unable to marshal request")

	body := make([]byte, 5+len(payload))
	binary.BigEndian.PutUint32(body[1:5], uint32(len(payload)))
	copy(body[5:], payload)

	daemonHTTPAddr := "http://127.0.0.1:" + port
	httpReq, err := http.NewRequest("POST", daemonHTTPAddr+"/EchoService/Echo", bytes.NewBuffer(body))
	require.NoError(t, err)

	httpReq.Header.Set("content-type", "application/grpc-web+proto")
	httpReq.Header.Set("snet-job-address", job.address)
	httpReq.Header.Set("snet-job-signature", job.signature)

	httpResp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)

	grpcMessage := httpResp.Header.Get("Grpc-Message")
	assert.Empty(t, grpcMessage, "Got non-empty Grpc-Message on response")

	grpcStatus := httpResp.Header.Get("Grpc-Status")
	assert.Empty(t, grpcStatus, "Got non-empty Grpc-Status code on response")

	httpRespBytes, err := ioutil.ReadAll(httpResp.Body)
	assert.NoError(t, err, "Unable to read HTTP response body from daemon")
	assert.NotEmpty(t, httpRespBytes, "Expected response body from daemon")
	resp := &echoservice.EchoEnvelope{}
	proto.Unmarshal(httpRespBytes[5:httpRespBytes[4]+5], resp)
	assert.Equal(t, resp.GetMessage(), "from gRPC-Web")
}
