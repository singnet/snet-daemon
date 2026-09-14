package cmd

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("routes grpc-web requests to the wrapped server", func(t *testing.T) {
		grpcWebServer := grpcweb.WrapServer(grpc.NewServer())
		handler := (&daemon{}).newHTTPHandler(grpcWebServer)

		req := httptest.NewRequest(http.MethodPost, "/some.Service/Method", nil)
		req.Header.Set("Content-Type", "application/grpc-web")
		response := httptest.NewRecorder()

		assert.NotPanics(t, func() {
			handler.ServeHTTP(response, req)
		})
		assert.NotEqual(t, http.StatusNotFound, response.Code)
	})

	t.Run("serves the heartbeat endpoint", func(t *testing.T) {
		originalEndpoint := config.GetString(config.ServiceEndpointKey)
		config.Vip().Set(config.ServiceEndpointKey, "http://127.0.0.1:1")
		t.Cleanup(func() {
			config.Vip().Set(config.ServiceEndpointKey, originalEndpoint)
		})

		components := &Components{
			serviceMetadata: &blockchain.ServiceMetadata{Encoding: "proto"},
			blockchain:      blockchain.NewMockProcessor(false),
			trainingService: &training.NoTrainingDaemonServer{},
			daemonHeartbeat: &metrics.DaemonHeartbeat{DynamicPricing: map[string]string{}},
		}
		handler := (&daemon{components: components}).newHTTPHandler(nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/heartbeat", nil))

		assert.Equal(t, http.StatusOK, response.Code)
		assert.NotEmpty(t, response.Body.String())
	})
}

func TestDaemonNewGRPCWebServer(t *testing.T) {
	d := &daemon{grpcServer: grpc.NewServer()}
	grpcWebServer := d.newGRPCWebServer()
	assert.NotNil(t, grpcWebServer)
}

func TestDaemonStartHTTP(t *testing.T) {
	originalDaemonType := config.GetString(config.DaemonTypeKey)
	config.Vip().Set(config.DaemonTypeKey, "http")
	t.Cleanup(func() {
		config.Vip().Set(config.DaemonTypeKey, originalDaemonType)
	})

	d := &daemon{
		lis:       testListener(t),
		blockProc: blockchain.NewMockProcessor(false),
	}

	assert.NotPanics(t, func() { d.start() })
	d.stop()
}

func TestDaemonStartGRPC(t *testing.T) {
	originalDaemonType := config.GetString(config.DaemonTypeKey)
	originalBlockchain := config.GetBool(config.BlockchainEnabledKey)
	config.Vip().Set(config.DaemonTypeKey, "grpc")
	config.Vip().Set(config.BlockchainEnabledKey, false)
	t.Cleanup(func() {
		config.Vip().Set(config.DaemonTypeKey, originalDaemonType)
		config.Vip().Set(config.BlockchainEnabledKey, originalBlockchain)
	})

	components := &Components{
		serviceMetadata: &blockchain.ServiceMetadata{Encoding: "proto"},
		blockchain:      blockchain.NewMockProcessor(false),
	}

	d := &daemon{
		lis:        testListener(t),
		blockProc:  blockchain.NewMockProcessor(false),
		components: components,
	}

	assert.NotPanics(t, func() { d.start() })
	d.stop()
}

func TestDaemonStartWithTLS(t *testing.T) {
	certPath, keyPath, cleanup := generateSelfSignedCert(t)
	defer cleanup()

	originalCert := config.GetString(config.SSLCertPathKey)
	originalKey := config.GetString(config.SSLKeyPathKey)
	originalDaemonType := config.GetString(config.DaemonTypeKey)
	config.Vip().Set(config.SSLCertPathKey, certPath)
	config.Vip().Set(config.SSLKeyPathKey, keyPath)
	config.Vip().Set(config.DaemonTypeKey, "http")
	t.Cleanup(func() {
		config.Vip().Set(config.SSLCertPathKey, originalCert)
		config.Vip().Set(config.SSLKeyPathKey, originalKey)
		config.Vip().Set(config.DaemonTypeKey, originalDaemonType)
	})

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)

	d := &daemon{
		lis:       testListener(t),
		blockProc: blockchain.NewMockProcessor(false),
		sslCert:   &cert,
	}

	assert.NotPanics(t, func() { d.start() })
	d.stop()
}

func TestDaemonStartWithTrafficSplit(t *testing.T) {
	originalEndpoint := config.GetString(config.DaemonEndpoint)
	config.Vip().Set(config.DaemonEndpoint, "127.0.0.1:8080")
	t.Cleanup(func() {
		config.Vip().Set(config.DaemonEndpoint, originalEndpoint)
	})

	d := &daemon{
		grpcServer:    grpc.NewServer(),
		autoSSLDomain: "example.com",
	}

	exp := &config.ExperimentalSettings{
		TrafficSplit: &struct {
			Grpc uint32 `json:"grpc"`
			Http uint32 `json:"http"`
		}{Grpc: 0, Http: 0},
	}

	assert.NotPanics(t, func() { d.startWithTrafficSplit(exp) })
	d.stop()
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
