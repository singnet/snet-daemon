package metrics

import (
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/singnet/snet-daemon/v6/training"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func acceptOnly(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return "http://" + ln.Addr().String()
}

func servingHTTP(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"SERVING"}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func noTraining() (*training.TrainingMetadata, error) { return nil, nil }
func noBlock() (*big.Int, error)                      { return big.NewInt(0), nil }

// Unconfigured heartbeat types must not report Online merely because a TCP
// listener accepts a connection (issue #694).
func TestGetHeartbeatUnconfiguredDoesNotClaimOnline(t *testing.T) {
	dead := acceptOnly(t)
	for _, hb := range []string{"", "none"} {
		t.Run("type_"+hbOrEmpty(hb), func(t *testing.T) {
			h, err := GetHeartbeat(dead, "", hb, "svc", noTraining, map[string]string{}, noBlock)
			require.NoError(t, err)
			assert.Equal(t, Warning.String(), h.Status,
				"unconfigured heartbeat must not look like a verified Online")
			assert.Contains(t, h.ServiceHeartbeat, `"status":"NOT_SERVING"`)
		})
	}
}

func hbOrEmpty(s string) string {
	if s == "" {
		return "empty"
	}
	return s
}

// Explicit tcp type still uses the TCP dial path.
func TestGetHeartbeatTCPStillPings(t *testing.T) {
	dead := acceptOnly(t)
	h, err := GetHeartbeat(dead, "", "tcp", "svc", noTraining, map[string]string{}, noBlock)
	require.NoError(t, err)
	assert.Equal(t, Online.String(), h.Status)
}

func TestGetHeartbeatHTTPOKRemainsOnline(t *testing.T) {
	url := servingHTTP(t)
	h, err := GetHeartbeat(url, url+"/heartbeat", "http", "svc", noTraining, map[string]string{}, noBlock)
	require.NoError(t, err)
	assert.Equal(t, Online.String(), h.Status)
}
