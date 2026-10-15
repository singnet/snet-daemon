package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type CertReloader struct {
	CertFile   string // path to the x509 certificate for https
	KeyFile    string // path to the x509 private key matching
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

func (cr *CertReloader) GetCertificate() *tls.Certificate {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	return cr.cachedCert
}

// TODO pass ctx here
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
