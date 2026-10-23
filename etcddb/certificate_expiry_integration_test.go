package etcddb

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func (suite *EtcdTestSuite) TestCertificateExpiryHTTPSLifecycle() {
	t := suite.T()
	now := time.Now()
	certPath, keyPath, _ := writeEtcdTestCertificateFromTemplate(t, &x509.Certificate{
		SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "local etcd TLS test"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage:    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	})
	setEtcdTLSPaths(t, certPath, keyPath, certPath)
	endpoint := startExpiryTestEtcd(t, certPath, keyPath)
	clientConf, err := json.Marshal(map[string]any{
		config.PaymentChannelStorageClientKey: map[string]any{
			"endpoints": []string{endpoint}, "connection_timeout": "5s", "request_timeout": "5s",
		},
	})
	require.NoError(t, err)
	core, logs := observer.New(zap.WarnLevel)
	t.Cleanup(zap.ReplaceGlobals(zap.New(core)))
	client, err := NewEtcdClientFromVip(readConfig(t, string(clientConf)), suite.metaData)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	require.NotNil(t, client.certificateMonitor)
	require.Equal(t, 1, logs.FilterMessageSnippet("certificate expires soon").Len(),
		"initialization must warn once even when the same certificate is in the client and CA files")
	require.NoError(t, client.Put("expiry-monitor-test", "connected over TLS"))
	value, ok, err := client.Get("expiry-monitor-test")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "connected over TLS", value)
	client.Close()
	select {
	case <-client.certificateMonitor.done:
	default:
		t.Fatal("closing a TLS client must stop its expiry monitor")
	}
}

func (suite *EtcdTestSuite) TestCertificateExpiryAlertBeforeFailedRequest() {
	t := suite.T()
	now := time.Now()
	certPath, keyPath, _ := writeEtcdTestCertificateFromTemplate(t, &x509.Certificate{
		SerialNumber: big.NewInt(43), NotBefore: now.Add(-2 * time.Hour), NotAfter: now.Add(-time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	})
	setEtcdTLSPaths(t, certPath, keyPath, certPath)
	core, logs := observer.New(zap.WarnLevel)
	t.Cleanup(zap.ReplaceGlobals(zap.New(core)))
	// Client initialization is asynchronous; the certificate warning must
	// still be emitted before a connection or storage request can succeed.
	vip := readConfig(t, `{"payment_channel_storage_client":{"endpoints":["https://127.0.0.1:1"],"request_timeout":"100ms"}}`)
	client, err := NewEtcdClientFromVip(vip, suite.metaData)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	alerts := logs.FilterMessageSnippet("certificate has expired").All()
	require.Len(t, alerts, 1)
	require.Equal(t, zap.ErrorLevel, alerts[0].Level)
	_, _, err = client.Get("unavailable-TLS-endpoint")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func startExpiryTestEtcd(t *testing.T, certPath, keyPath string) string {
	t.Helper()
	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	cfg.LogLevel, cfg.LogOutputs = "error", []string{"stderr"}
	cfg.ListenClientUrls = []url.URL{{Scheme: "https", Host: "127.0.0.1:0"}}
	cfg.AdvertiseClientUrls = cfg.ListenClientUrls
	cfg.ListenPeerUrls = []url.URL{{Scheme: "http", Host: "127.0.0.1:0"}}
	cfg.AdvertisePeerUrls = cfg.ListenPeerUrls
	cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)
	cfg.ClientTLSInfo = transport.TLSInfo{
		CertFile: certPath, KeyFile: keyPath, TrustedCAFile: certPath, ClientCertAuth: true,
	}
	server, err := embed.StartEtcd(cfg)
	require.NoError(t, err)
	t.Cleanup(server.Close)
	select {
	case <-server.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		t.Fatal("local TLS etcd did not become ready")
	}
	require.Len(t, server.Clients, 1)
	return "https://" + server.Clients[0].Addr().String()
}
