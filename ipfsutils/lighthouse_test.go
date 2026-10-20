package ipfsutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLighthouseFile_Success(t *testing.T) {
	block := unixFSFixture(t, []byte("filecoin-content"))
	serveBlocks(t, true, block)
	data, err := GetLighthouseFile(block.cid.String())
	require.NoError(t, err)
	assert.Equal(t, "filecoin-content", string(data))
}

func TestGetLighthouseFile_HTTPError(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{config.LighthouseEndpoint: ts.URL + "/"})

	data, err := GetLighthouseFile(rawFixture(t, []byte("file")).cid.String())
	require.ErrorContains(t, err, "HTTP 500: server error")
	assert.Nil(t, data)
}

func TestGetLighthouseFile_ConnectionRefused(t *testing.T) {
	withTestConfig(t, map[string]string{config.LighthouseEndpoint: "http://127.0.0.1:1/"})

	data, err := GetLighthouseFile(rawFixture(t, []byte("file")).cid.String())
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetLighthouseFile_ReadBodyError(t *testing.T) {
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

	withTestConfig(t, map[string]string{config.LighthouseEndpoint: ts.URL + "/"})

	data, err := GetLighthouseFile(rawFixture(t, []byte("file")).cid.String())
	assert.Error(t, err)
	assert.Nil(t, data)
}

// ========== common.go (ReadFile) ==========
