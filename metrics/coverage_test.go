package metrics

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/training"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestTCPPingService(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := lis.Addr().String()

	t.Run("http with explicit port", func(t *testing.T) {
		assert.NoError(t, tcpPingService("http://"+addr))
	})
	t.Run("no scheme defaults to http", func(t *testing.T) {
		assert.NoError(t, tcpPingService(addr))
	})
	t.Run("https with explicit port", func(t *testing.T) {
		assert.NoError(t, tcpPingService("https://"+addr))
	})
	t.Run("connection refused", func(t *testing.T) {
		assert.Error(t, tcpPingService("http://127.0.0.1:1"))
	})
}

func TestCallRegisterService(t *testing.T) {
	isolatedMetricsConfig(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"success","data":{"token":"generated-token"}}`)
	}))
	defer server.Close()

	status := callRegisterService("some-daemon-id", server.URL)
	assert.True(t, status)
	assert.Equal(t, "generated-token", daemonAuthorizationToken)
}

func TestCallRegisterServiceErrors(t *testing.T) {
	isolatedMetricsConfig(t)

	t.Run("unreachable service", func(t *testing.T) {
		status := callRegisterService("some-daemon-id", "http://127.0.0.1:1/register")
		assert.False(t, status)
	})

	t.Run("malformed url", func(t *testing.T) {
		status := callRegisterService("some-daemon-id", "://bad-url")
		assert.False(t, status)
	})
}

func TestBuildPayLoadForServiceRegistration(t *testing.T) {
	isolatedMetricsConfig(t)

	payload := buildPayLoadForServiceRegistration()
	var parsed RegisterDaemonPayload
	require.NoError(t, json.Unmarshal(payload, &parsed))
	assert.NotEmpty(t, parsed.DaemonID)
}

func TestRegisterDaemon(t *testing.T) {
	isolatedMetricsConfig(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"success","data":{"token":"generated-token"}}`)
	}))
	defer server.Close()

	assert.True(t, RegisterDaemon(server.URL))
}

func TestRegisterDaemonFailure(t *testing.T) {
	isolatedMetricsConfig(t)
	assert.False(t, RegisterDaemon("http://127.0.0.1:1/register"))
}

func TestStatusStringUnknown(t *testing.T) {
	assert.Equal(t, "Unknown", Status(99).String())
	assert.Equal(t, "Unknown", Status(-1).String())
}

func TestValidateHeartbeatConfigTypes(t *testing.T) {
	for _, typ := range []string{"", "none", "http", "https", "grpc", "tcp"} {
		assert.NoError(t, ValidateHeartbeatConfig(typ, "http://example.test"), "type %q should be valid", typ)
	}

	assert.NoError(t, ValidateHeartbeatConfig("none", ""), "empty endpoint with none type should be valid")

	assert.Error(t, ValidateHeartbeatConfig("unknown", "http://example.test"))
	assert.Error(t, ValidateHeartbeatConfig("http", "not a valid url"))
}

func TestGetStorageCertificateDetailsEmptyConfig(t *testing.T) {
	isolatedMetricsConfig(t)
	cert := getStorageCertificateDetails()
	assert.Equal(t, StorageClientCert{}, cert)
}

func generateSelfSignedCertFiles(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	certOut, err := os.CreateTemp(t.TempDir(), "cert*.pem")
	require.NoError(t, err)
	defer certOut.Close()
	require.NoError(t, pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}))

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyOut, err := os.CreateTemp(t.TempDir(), "key*.pem")
	require.NoError(t, err)
	defer keyOut.Close()
	require.NoError(t, pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}))

	return certOut.Name(), keyOut.Name()
}

func TestGetStorageCertificateDetailsWithCert(t *testing.T) {
	isolatedMetricsConfig(t)
	certPath, keyPath := generateSelfSignedCertFiles(t)
	config.Vip().Set(config.PaymentChannelCertPath, certPath)
	config.Vip().Set(config.PaymentChannelKeyPath, keyPath)

	cert := getStorageCertificateDetails()
	assert.NotEmpty(t, cert.ValidFrom)
	assert.NotEmpty(t, cert.ValidTill)
}

func TestCallHTTPServiceHeartbeatEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	body, err := callHTTPServiceHeartbeat(server.URL)
	assert.Error(t, err)
	assert.Nil(t, body)
}

func TestDaemonHeartbeatList(t *testing.T) {
	resp, err := (&DaemonHeartbeat{}).List(context.Background(), &grpc_health_v1.HealthListRequest{})
	assert.Nil(t, resp)
	assert.Error(t, err)
}

func TestDaemonHeartbeatWatch(t *testing.T) {
	assert.NoError(t, (&DaemonHeartbeat{}).Watch(nil, nil))
}

func startGrpcHealthServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(server, &clientImplHeartBeat{})
	go server.Serve(lis)
	t.Cleanup(server.Stop)
	return lis.Addr().String()
}

func startTCPServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	t.Cleanup(func() { lis.Close() })
	return lis.Addr().String()
}

func testCurrentBlock() func() (*big.Int, error) {
	return func() (*big.Int, error) { return big.NewInt(100), nil }
}

func TestGetHeartbeatGrpc(t *testing.T) {
	isolatedMetricsConfig(t)
	addr := startGrpcHealthServer(t)

	hb, err := GetHeartbeat("http://127.0.0.1:1", addr, "grpc", "SERVICE001", nil, nil, testCurrentBlock())
	assert.NoError(t, err)
	assert.Equal(t, Online.String(), hb.Status)
}

func TestGetHeartbeatTCP(t *testing.T) {
	isolatedMetricsConfig(t)
	addr := startTCPServer(t)

	for _, typ := range []string{"", "none", "tcp"} {
		hb, err := GetHeartbeat("http://"+addr, "", typ, "SERVICE001", nil, nil, testCurrentBlock())
		assert.NoError(t, err)
		assert.Equal(t, Online.String(), hb.Status)
	}
}

func TestDaemonHeartbeatCheck(t *testing.T) {
	isolatedMetricsConfig(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"serviceID":"SERVICE001","status":"SERVING"}`)
	}))
	defer server.Close()

	config.Vip().Set(config.ServiceHeartbeatType, "http")
	config.Vip().Set(config.HeartbeatServiceEndpoint, server.URL+"/heartbeat")
	config.Vip().Set(config.ServiceId, "SERVICE001")

	h := &DaemonHeartbeat{
		TrainingMetadata: func() (*training.TrainingMetadata, error) { return &training.TrainingMetadata{}, nil },
		CurrentBlock:     testCurrentBlock(),
	}

	resp, err := h.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestDaemonHeartbeatCheckUnavailable(t *testing.T) {
	isolatedMetricsConfig(t)

	config.Vip().Set(config.ServiceHeartbeatType, "http")
	config.Vip().Set(config.HeartbeatServiceEndpoint, "http://127.0.0.1:1/heartbeat")

	h := &DaemonHeartbeat{
		TrainingMetadata: func() (*training.TrainingMetadata, error) { return &training.TrainingMetadata{}, nil },
		CurrentBlock:     testCurrentBlock(),
	}

	resp, err := h.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	assert.Error(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN, resp.Status)
}

func TestGetStatus(t *testing.T) {
	assert.Equal(t, "success", getStatus(nil))
	assert.Equal(t, "failed", getStatus(fmt.Errorf("boom")))
}

func TestPublishResponseStats(t *testing.T) {
	isolatedMetricsConfig(t)
	config.Vip().Set(config.MeteringEndpoint, "https://metering.example.test")

	calls := 0
	stubMetricsHTTP(t, func(req *http.Request) (*http.Response, error) {
		calls++
		assert.Equal(t, http.MethodPost, req.Method)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	stats := BuildCommonStats(time.Now(), "TestMethod")
	ok := PublishResponseStats(stats, time.Second, nil, big.NewInt(100))
	assert.True(t, ok)
	assert.Equal(t, 1, calls)
}

func TestNotificationSendConfigured(t *testing.T) {
	isolatedMetricsConfig(t)
	SetIsNoAlertsConfig(false)
	config.Vip().Set(config.NotificationServiceEndpoint, "https://notify.example.test")

	calls := 0
	stubMetricsHTTP(t, func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	alert := &Notification{
		Recipient: "alerts@example.com",
		Message:   "some message",
		Details:   "details",
		Component: "Daemon",
		DaemonID:  "daemon-id",
		Level:     "ERROR",
	}
	result := alert.Send(big.NewInt(100))
	assert.True(t, result)
	assert.Equal(t, 1, calls)
}

func TestValidateNotificationConfigValid(t *testing.T) {
	isolatedMetricsConfig(t)
	config.Vip().Set(config.NotificationServiceEndpoint, "https://notify.example.test")
	config.Vip().Set(config.AlertsEMail, "alerts@example.com")

	err := ValidateNotificationConfig()
	assert.NoError(t, err)
	assert.False(t, isNoAlertsConfig)
}

func TestValidateNotificationConfigInvalid(t *testing.T) {
	isolatedMetricsConfig(t)
	config.Vip().Set(config.NotificationServiceEndpoint, "not-a-valid-url")
	config.Vip().Set(config.AlertsEMail, "alerts@example.com")

	err := ValidateNotificationConfig()
	assert.Error(t, err)
}
