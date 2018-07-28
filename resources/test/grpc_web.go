package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGRPCWeb(t *testing.T, h harness, port string) {
	job := h.createSignedJob()
	time.Sleep(time.Second)

	daemonHTTPAddr := "http://127.0.0.1:" + port
	httpReq, err := http.NewRequest("POST", daemonHTTPAddr+"/FakeService/FakeMethod",
		bytes.NewBuffer([]byte("\x00\x00\x00\x00\x13"+`{"hello":"goodbye"}`)))
	require.NoError(t, err)

	httpReq.Header.Set("content-type", "application/grpc-web+json")
	httpReq.Header.Set("snet-job-address", job.address)
	httpReq.Header.Set("snet-job-signature", job.signature)

	httpResp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)

	grpcMessage := httpResp.Header.Get("Grpc-Message")
	assert.Empty(t, grpcMessage, "Got non-empty Grpc-Message on response")

	grpcStatus := httpResp.Header.Get("Grpc-Status")
	assert.Empty(t, grpcStatus, "Got non-empty Grpc-Status code on response")

	httpRespBytes, err := ioutil.ReadAll(httpResp.Body)
	assert.NoError(t, err, "Unable to read HTP response body from daemon")
	fmt.Print(string(httpRespBytes))
	assert.NotEmpty(t, httpRespBytes, "Expected response body from daemon")
}
