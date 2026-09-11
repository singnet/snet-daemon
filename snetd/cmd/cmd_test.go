package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFileExist(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("existing file", func(t *testing.T) {
		f := filepath.Join(tmpDir, "exists.txt")
		require.NoError(t, os.WriteFile(f, []byte("data"), 0644))
		assert.True(t, isFileExist(f))
	})

	t.Run("non-existent file", func(t *testing.T) {
		assert.False(t, isFileExist(filepath.Join(tmpDir, "nope.txt")))
	})

	t.Run("directory", func(t *testing.T) {
		assert.True(t, isFileExist(tmpDir))
	})
}

func TestLoadConfigFileFromCommandLine(t *testing.T) {
	t.Run("existing config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "test_config.json")
		require.NoError(t, os.WriteFile(cfgPath, []byte(`{"daemon_endpoint_port":12345}`), 0644))

		cmd := &cobra.Command{}
		cmd.Flags().String("config", cfgPath, "")
		cmd.Flags().Set("config", cfgPath)

		flag := cmd.Flags().Lookup("config")
		assert.NotPanics(t, func() {
			loadConfigFileFromCommandLine(flag)
		})
	})

	t.Run("config file not found panics", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "/nonexistent/config.json", "")
		cmd.Flags().Set("config", "/nonexistent/config.json")

		flag := cmd.Flags().Lookup("config")
		assert.Panics(t, func() {
			loadConfigFileFromCommandLine(flag)
		})
	})

	t.Run("default config file not set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("config", "snetd.config.json", "")

		flag := cmd.Flags().Lookup("config")
		assert.NotPanics(t, func() {
			loadConfigFileFromCommandLine(flag)
		})
	})
}

func generateSelfSignedCert(t *testing.T) (certPath, keyPath string, cleanup func()) {
	t.Helper()
	tmpDir := t.TempDir()
	certPath = filepath.Join(tmpDir, "cert.pem")
	keyPath = filepath.Join(tmpDir, "key.pem")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	certFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	keyFile.Close()

	cleanup = func() {
		os.Remove(certPath)
		os.Remove(keyPath)
	}
	return certPath, keyPath, cleanup
}

func TestCertReloaderReloadCertificate(t *testing.T) {
	certPath, keyPath, cleanup := generateSelfSignedCert(t)
	defer cleanup()

	cr := &CertReloader{
		CertFile: certPath,
		KeyFile:  keyPath,
		mutex:    &sync.Mutex{},
	}

	err := cr.reloadCertificate()
	assert.NoError(t, err)
	assert.NotNil(t, cr.GetCertificate())
}

func TestCertReloaderReloadCertificateInvalidFiles(t *testing.T) {
	cr := &CertReloader{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
		mutex:    &sync.Mutex{},
	}

	err := cr.reloadCertificate()
	assert.Error(t, err)
}

func TestCertReloaderGetCertificateNil(t *testing.T) {
	cr := &CertReloader{
		mutex: &sync.Mutex{},
	}
	assert.Nil(t, cr.GetCertificate())
}

func TestCertReloaderListen(t *testing.T) {
	certPath, keyPath, cleanup := generateSelfSignedCert(t)
	defer cleanup()

	cr := &CertReloader{
		CertFile: certPath,
		KeyFile:  keyPath,
		mutex:    &sync.Mutex{},
	}

	cr.Listen()

	// Wait for the goroutine to reload the certificate
	time.Sleep(4 * time.Second)
	assert.NotNil(t, cr.GetCertificate())
}

func TestCheckResponseNil(t *testing.T) {
	allowed, err := checkResponse(nil)
	assert.False(t, allowed)
	assert.EqualError(t, err, "Empty response received.")
}

func TestCheckResponseNonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	allowed, err := checkResponse(resp)
	assert.False(t, allowed)
	assert.Contains(t, err.Error(), "Service call failed with status code : 500")
}

func TestCheckResponseSuccess(t *testing.T) {
	body := `{"data":"success"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer ts.Close()

	httpResp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	allowed, err := checkResponse(httpResp)
	assert.True(t, allowed)
	assert.NoError(t, err)
}

func TestCheckResponseFailedData(t *testing.T) {
	body := `{"data":"failed"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer ts.Close()

	httpResp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	config.Vip().Set(config.MeteringEndpoint, ts.URL)
	defer config.Vip().Set(config.MeteringEndpoint, "")

	allowed, err := checkResponse(httpResp)
	assert.False(t, allowed)
	assert.Contains(t, err.Error(), "error returned by by Metering Service")
}

func TestCheckResponseInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer ts.Close()

	httpResp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	allowed, err := checkResponse(httpResp)
	assert.False(t, allowed)
	assert.Error(t, err)
}

type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) { return 0, assert.AnError }
func (e *errReader) Close() error                     { return nil }

func TestCheckResponseReadBodyError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       &errReader{},
	}
	allowed, err := checkResponse(resp)
	assert.False(t, allowed)
	assert.Error(t, err)
}

func TestGetPaymentChannelIdEmpty(t *testing.T) {
	id, err := getPaymentChannelId(&cobra.Command{})
	assert.NoError(t, err)
	assert.Nil(t, id)
}

func TestGetPaymentChannelIdValid(t *testing.T) {
	paymentChannelId = "123"
	defer func() { paymentChannelId = "" }()

	id, err := getPaymentChannelId(&cobra.Command{})
	assert.NoError(t, err)
	assert.NotNil(t, id)
	assert.Equal(t, big.NewInt(123), id)
}

func TestGetPaymentChannelIdInvalid(t *testing.T) {
	paymentChannelId = "not-a-number"
	defer func() { paymentChannelId = "" }()

	id, err := getPaymentChannelId(&cobra.Command{})
	assert.Error(t, err)
	assert.Nil(t, id)
	assert.Contains(t, err.Error(), "Incorrect decimal number format")
}

func TestGetFreeCallIDs(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP(AddressFlag, "a", "", "")
	cmd.Flags().StringP(UserIdFlag, "u", "", "")
	cmd.Flags().Set(AddressFlag, "0x1234")
	cmd.Flags().Set(UserIdFlag, "user@example.com")

	userId, address, err := getFreeCallIDs(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "0x1234", address)
	assert.Equal(t, "user@example.com", userId)
}

func TestGetFreeCallIDsAddressOnly(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP(AddressFlag, "a", "", "")
	cmd.Flags().StringP(UserIdFlag, "u", "", "")
	cmd.Flags().Set(AddressFlag, "0x5678")

	userId, address, err := getFreeCallIDs(cmd)
	assert.NoError(t, err)
	assert.Equal(t, "0x5678", address)
	assert.Equal(t, "", userId)
}

func TestComponentsCloseNilFields(t *testing.T) {
	components := &Components{}
	assert.NotPanics(t, func() {
		components.Close()
	})
}
