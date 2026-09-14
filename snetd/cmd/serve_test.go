package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// TODO
func TestDaemonPort(t *testing.T) {
	assert.Equal(t, config.GetString(config.DaemonEndpoint), "127.0.0.1:8080")
}

func TestNewDaemonRejectsInvalidConfiguration(t *testing.T) {
	originalDaemonType := config.GetString(config.DaemonTypeKey)
	config.Vip().Set(config.DaemonTypeKey, "invalid")
	t.Cleanup(func() {
		config.Vip().Set(config.DaemonTypeKey, originalDaemonType)
	})

	_, err := newDaemon(&Components{})
	assert.EqualError(t, err, "unrecognized DAEMON_TYPE 'invalid'")
}

func TestDaemonNewHTTPHandler(t *testing.T) {
	t.Run("handles CORS preflight", func(t *testing.T) {
		handler := (&daemon{}).newHTTPHandler(nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodOptions, "/anything", nil))

		assert.Equal(t, http.StatusNoContent, response.Code)
	})

	t.Run("returns not found for an unsupported path", func(t *testing.T) {
		handler := (&daemon{}).newHTTPHandler(nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/unknown", nil))

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns the service wire encoding", func(t *testing.T) {
		handler := (&daemon{components: &Components{
			serviceMetadata: &blockchain.ServiceMetadata{Encoding: "proto"},
		}}).newHTTPHandler(nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/encoding", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "proto\n", response.Body.String())
	})
}

func TestDaemonStopClosesListeners(t *testing.T) {
	listener := testListener(t)
	acmeListener := testListener(t)
	d := &daemon{
		grpcServer:   grpc.NewServer(),
		lis:          listener,
		acmeListener: acmeListener,
	}

	d.stop()

	assert.Error(t, listener.Close())
	assert.Error(t, acmeListener.Close())
}
