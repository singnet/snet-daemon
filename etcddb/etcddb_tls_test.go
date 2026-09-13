package etcddb

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
)

func TestGetTLSConfig(t *testing.T) {
	certPath, keyPath, certDER := writeEtcdTestCertificate(t)
	missingPath := filepath.Join(t.TempDir(), "missing.pem")
	for _, tc := range []struct {
		name, cert, key, ca, errorText string
	}{
		{"valid certificate and CA", certPath, keyPath, certPath, ""},
		{"missing certificate", missingPath, keyPath, certPath, "load x509 keypair"},
		{"missing private key", certPath, missingPath, certPath, "load x509 keypair"},
		{"missing CA", certPath, keyPath, missingPath, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setEtcdTLSPaths(t, tc.cert, tc.key, tc.ca)
			cfg, err := getTLSConfig()
			if tc.cert == missingPath || tc.key == missingPath || tc.ca == missingPath {
				require.ErrorIs(t, err, os.ErrNotExist)
				if tc.errorText != "" {
					require.ErrorContains(t, err, tc.errorText)
				}
				require.Nil(t, cfg)
				return
			}
			require.NoError(t, err)
			require.Len(t, cfg.Certificates, 1)
			require.Equal(t, [][]byte{certDER}, cfg.Certificates[0].Certificate)
			require.NotNil(t, cfg.Certificates[0].PrivateKey)
			require.NotNil(t, cfg.RootCAs)
			cert, err := x509.ParseCertificate(certDER)
			require.NoError(t, err)
			_, err = cert.Verify(x509.VerifyOptions{
				Roots: cfg.RootCAs, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			})
			require.NoError(t, err, "the supplied CA must be trusted by the returned TLS config")
		})
	}
}

func (suite *EtcdTestSuite) TestHTTPSClientRejectsMissingCertificate() {
	t := suite.T()
	missingPath := filepath.Join(t.TempDir(), "missing.pem")
	setEtcdTLSPaths(t, missingPath, missingPath, missingPath)
	vip := readConfig(t, `{"payment_channel_storage_client":{"endpoints":["https://127.0.0.1:2379"]}}`)
	client, err := NewEtcdClientFromVip(vip, suite.metaData)
	require.ErrorContains(t, err, "load x509 keypair")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Nil(t, client)
}

func setEtcdTLSPaths(t *testing.T, cert, key, ca string) {
	t.Helper()
	for name, value := range map[string]string{
		config.PaymentChannelCertPath: cert,
		config.PaymentChannelKeyPath:  key,
		config.PaymentChannelCaPath:   ca,
	} {
		t.Setenv("SNET_"+strings.ToUpper(name), value)
	}
}

func writeEtcdTestCertificate(t *testing.T) (certPath, keyPath string, certDER []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "etcd test client"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage:    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDER, err = x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	dir := t.TempDir()
	certPath, keyPath = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600))
	return
}
