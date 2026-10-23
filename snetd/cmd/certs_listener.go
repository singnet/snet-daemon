package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CertReloader caches a PEM certificate/key pair for subsequent TLS handshakes.
type CertReloader struct {
	CertFile   string // Path to the PEM-encoded X.509 certificate chain.
	KeyFile    string // Path to the matching PEM-encoded private key.
	mutex      *sync.Mutex
	cachedCert *tls.Certificate
}

func (cr *CertReloader) reloadCertificate() error {
	pair, err := tls.LoadX509KeyPair(cr.CertFile, cr.KeyFile)
	if err != nil {
		return fmt.Errorf("failed loading tls key pair: %w", err)
	}
	cr.mutex.Lock()
	cr.cachedCert = &pair
	cr.mutex.Unlock()
	return err
}

// GetCertificate returns the last successfully reloaded pair, or nil before the first reload.
func (cr *CertReloader) GetCertificate() *tls.Certificate {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	return cr.cachedCert
}

// Listen reloads the pair every three seconds, keeping the previous pair on errors.
// TODO: Accept a context so the polling goroutine can stop with the daemon.
func (cr *CertReloader) Listen() {
	go cr.ListenContext(context.Background())
}

func (cr *CertReloader) ListenContext(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := cr.reloadCertificate()
			if err != nil {
				zap.L().Error("Error in reloading ssl certificates", zap.Error(err))
			}
		}
	}
}
