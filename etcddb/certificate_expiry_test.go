package etcddb

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func expiryTestCertificate(notAfter time.Time) monitoredCertificate {
	return monitoredCertificate{
		certificate: &x509.Certificate{
			Raw: []byte("test-certificate"), NotAfter: notAfter,
			Subject: pkix.Name{CommonName: "etcd-client"},
		},
		path: "client.pem", role: "client",
	}
}

func TestCertificateExpiryThresholds(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		remaining time.Duration
		wantCount int
		wantLevel zapcore.Level
		wantState string
	}{
		{"healthy", certificateExpiryWarningWindow + time.Nanosecond, 0, zap.WarnLevel, ""},
		{"warning boundary", certificateExpiryWarningWindow, 1, zap.WarnLevel, "expires soon"},
		{"nearly expired", time.Nanosecond, 1, zap.WarnLevel, "expires soon"},
		{"expiry boundary", 0, 1, zap.ErrorLevel, "has expired"},
		{"expired", -time.Hour, 1, zap.ErrorLevel, "has expired"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			var hookMessages []string
			logger := zap.New(core, zap.Hooks(func(entry zapcore.Entry) error {
				hookMessages = append(hookMessages, entry.Message)
				return nil
			}))
			cert := expiryTestCertificate(now.Add(tc.remaining))
			monitor := newCertificateExpiryMonitor([]monitoredCertificate{cert}, logger)
			monitor.check(now)
			require.Equal(t, tc.wantCount, logs.Len())
			require.Len(t, hookMessages, tc.wantCount)
			if tc.wantCount == 0 {
				return
			}
			entry := logs.All()[0]
			require.Equal(t, tc.wantLevel, entry.Level)
			require.Equal(t, entry.Message, hookMessages[0])
			for _, expected := range []string{
				"[etcd] client certificate", tc.wantState, "client.pem", "CN=etcd-client",
				"SHA256=", cert.certificate.NotAfter.UTC().Format(time.RFC3339),
				"recreate the etcd client", "remaining=",
			} {
				require.Contains(t, hookMessages[0], expected)
			}
			require.NotContains(t, entry.Message, string(cert.certificate.Raw))
		})
	}
}

func TestCertificateExpiryDailyWarnings(t *testing.T) {
	now := time.Now()
	core, logs := observer.New(zap.DebugLevel)
	monitor := newCertificateExpiryMonitor([]monitoredCertificate{
		expiryTestCertificate(now.Add(certificateExpiryWarningWindow + time.Hour)),
	}, zap.New(core))
	monitor.check(now)
	require.Zero(t, logs.Len())
	monitor.check(now.Add(time.Hour))
	require.Equal(t, 1, logs.Len())
	monitor.check(now.Add(time.Hour + certificateExpiryRepeatInterval - time.Nanosecond))
	require.Equal(t, 1, logs.Len())
	monitor.check(now.Add(time.Hour + certificateExpiryRepeatInterval))
	require.Equal(t, 2, logs.Len())
	for _, entry := range logs.All() {
		require.Equal(t, zap.WarnLevel, entry.Level)
	}
}

func TestCertificateExpiryEscalationAndErrorReminders(t *testing.T) {
	now := time.Now()
	core, logs := observer.New(zap.DebugLevel)
	monitor := newCertificateExpiryMonitor([]monitoredCertificate{
		expiryTestCertificate(now.Add(time.Hour)),
	}, zap.New(core))
	monitor.check(now)
	monitor.check(now.Add(time.Minute))
	require.Equal(t, 1, logs.Len())
	monitor.check(now.Add(time.Hour))
	require.Equal(t, 2, logs.Len(), "expiry must not be suppressed by the recent warning")
	require.Equal(t, zap.ErrorLevel, logs.All()[1].Level)
	monitor.check(now.Add(time.Hour + certificateExpiryRepeatInterval - time.Nanosecond))
	require.Equal(t, 2, logs.Len())
	monitor.check(now.Add(time.Hour + certificateExpiryRepeatInterval))
	require.Equal(t, 3, logs.Len())
	require.Equal(t, zap.ErrorLevel, logs.All()[2].Level)
}

func TestCertificateExpiryDeduplicatesChainAndCA(t *testing.T) {
	now := time.Now()
	client := expiryTestCertificate(now.Add(time.Hour))
	ca := client
	ca.path, ca.role = "ca.pem", "CA"
	core, logs := observer.New(zap.DebugLevel)
	monitor := newCertificateExpiryMonitor([]monitoredCertificate{client, ca, ca}, zap.New(core))
	monitor.check(now)
	require.Equal(t, 1, logs.Len())
	monitor.check(now.Add(certificateExpiryRepeatInterval))
	require.Equal(t, 2, logs.Len())
}

func TestCertificateExpiryDoesNotSuppressOtherCertificates(t *testing.T) {
	now := time.Now()
	client := expiryTestCertificate(now.Add(2 * certificateExpiryWarningWindow))
	chain := expiryTestCertificate(now.Add(time.Hour))
	chain.certificate.Raw, chain.role = []byte("intermediate"), "client chain"
	ca := expiryTestCertificate(now.Add(-time.Hour))
	ca.certificate.Raw, ca.path, ca.role = []byte("CA"), "ca.pem", "CA"
	core, logs := observer.New(zap.DebugLevel)
	monitor := newCertificateExpiryMonitor([]monitoredCertificate{client, chain, ca}, zap.New(core))
	monitor.check(now)
	require.Equal(t, 2, logs.Len())
	require.Equal(t, zap.WarnLevel, logs.All()[0].Level)
	require.Contains(t, logs.All()[0].Message, "client chain certificate")
	require.Equal(t, zap.ErrorLevel, logs.All()[1].Level)
	require.Contains(t, logs.All()[1].Message, "CA certificate")
}

func TestCertificateExpiryDoesNotThrottleDisabledLogLevel(t *testing.T) {
	now := time.Now()
	level := zap.NewAtomicLevelAt(zap.ErrorLevel)
	core, logs := observer.New(zap.DebugLevel)
	monitor := newCertificateExpiryMonitor([]monitoredCertificate{expiryTestCertificate(now.Add(time.Hour))},
		zap.New(core, zap.IncreaseLevel(level)))
	monitor.check(now)
	require.Zero(t, logs.Len())
	level.SetLevel(zap.WarnLevel)
	monitor.check(now.Add(time.Minute))
	require.Equal(t, 1, logs.Len())
}

func TestCertificateExpiryPeriodicChecksStopWithClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		now := time.Now()
		core, logs := observer.New(zap.DebugLevel)
		monitor := newCertificateExpiryMonitor([]monitoredCertificate{
			expiryTestCertificate(now.Add(certificateExpiryWarningWindow + 2*time.Hour)),
		}, zap.New(core))
		monitor.check(now)
		require.Zero(t, logs.Len())
		monitor.start()
		client := &EtcdClient{certificateMonitor: monitor}
		t.Cleanup(client.Close)
		synctest.Wait()
		time.Sleep(2 * time.Hour)
		synctest.Wait()
		require.Equal(t, 1, logs.Len(), "the timer must warn without a heartbeat request")
		client.Close()
		client.Close()
		select {
		case <-monitor.done:
		default:
			t.Fatal("Close must wait for the certificate monitor to stop")
		}
		time.Sleep(2 * certificateExpiryRepeatInterval)
		require.Equal(t, 1, logs.Len(), "closed clients must not keep generating alerts")
	})
}

func TestGetTLSConfigMonitorsWholeChainAndCABundle(t *testing.T) {
	certPath, keyPath, leafDER := writeEtcdTestCertificate(t)
	caPath, _, caDER := writeEtcdTestCertificate(t)
	_, _, intermediateDER := writeEtcdTestCertificate(t)
	chain := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intermediateDER})...)
	require.NoError(t, os.WriteFile(certPath, chain, 0600))
	caBundle := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intermediateDER})...)
	for _, block := range []*pem.Block{
		{Type: "OTHER", Bytes: []byte("ignored")},
		{Type: "CERTIFICATE", Bytes: []byte("invalid DER")},
		{Type: "CERTIFICATE", Bytes: caDER, Headers: map[string]string{"ignored": "header"}},
	} {
		caBundle = append(caBundle, pem.EncodeToMemory(block)...)
	}
	caBundle = append(caBundle, []byte("not a PEM block")...)
	require.NoError(t, os.WriteFile(caPath, caBundle, 0600))
	setEtcdTLSPaths(t, certPath, keyPath, caPath)
	cfg, certificates, err := getTLSConfig()
	require.NoError(t, err)
	require.Len(t, cfg.Certificates[0].Certificate, 2)
	require.Len(t, certificates, 4)
	for i, expected := range []struct {
		der  []byte
		role string
		path string
	}{
		{leafDER, "client", certPath}, {intermediateDER, "client chain", certPath},
		{caDER, "CA", caPath}, {intermediateDER, "CA", caPath},
	} {
		require.Equal(t, expected.der, certificates[i].certificate.Raw)
		require.Equal(t, expected.role, certificates[i].role)
		require.Equal(t, expected.path, certificates[i].path)
	}
	core, logs := observer.New(zap.DebugLevel)
	newCertificateExpiryMonitor(certificates, zap.New(core)).check(time.Now())
	require.Equal(t, 3, logs.Len(), "the intermediate in both files must only generate one warning")
}

func TestCertificateExpiryMonitorsLoadedCertificatesUntilRecreated(t *testing.T) {
	certPath, keyPath, originalDER := writeEtcdTestCertificate(t)
	setEtcdTLSPaths(t, certPath, keyPath, certPath)
	_, loaded, err := getTLSConfig()
	require.NoError(t, err)
	core, logs := observer.New(zap.DebugLevel)
	monitor := newCertificateExpiryMonitor(loaded, zap.New(core))
	originalExpiry := loaded[0].certificate.NotAfter

	renewedCert, renewedKey, renewedDER := writeEtcdTestCertificateFromTemplate(t, &x509.Certificate{
		SerialNumber: loaded[0].certificate.SerialNumber,
		NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(90 * 24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	})
	for destination, source := range map[string]string{certPath: renewedCert, keyPath: renewedKey} {
		data, err := os.ReadFile(source)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(destination, data, 0600))
	}
	monitor.check(originalExpiry)
	require.Equal(t, 1, logs.Len())
	require.Equal(t, zap.ErrorLevel, logs.All()[0].Level)
	require.Equal(t, originalDER, loaded[0].certificate.Raw)
	_, reloaded, err := getTLSConfig()
	require.NoError(t, err)
	require.Equal(t, renewedDER, reloaded[0].certificate.Raw)
	newCertificateExpiryMonitor(reloaded, zap.New(core)).check(originalExpiry)
	require.Equal(t, 1, logs.Len(), "a recreated client must monitor the renewed certificate")
}
