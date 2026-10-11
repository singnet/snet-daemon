package etcddb

import (
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	certificateExpiryWarningWindow  = 30 * 24 * time.Hour
	certificateExpiryCheckInterval  = time.Hour
	certificateExpiryRepeatInterval = 24 * time.Hour
)

// monitoredCertificate describes a certificate loaded into this client's TLS
// configuration, not the current contents of a potentially replaced PEM file.
type monitoredCertificate struct {
	certificate *x509.Certificate
	path        string
	role        string
}

type certificateExpiryAlert struct {
	level zapcore.Level
	time  time.Time
}

type certificateExpiryMonitor struct {
	certificates []monitoredCertificate
	logger       *zap.Logger
	alerts       map[[sha256.Size]byte]certificateExpiryAlert
	stop         chan struct{}
	done         chan struct{}
	stopOnce     sync.Once
}

func newCertificateExpiryMonitor(certificates []monitoredCertificate, logger *zap.Logger) *certificateExpiryMonitor {
	return &certificateExpiryMonitor{
		certificates: certificates,
		logger:       logger,
		alerts:       make(map[[sha256.Size]byte]certificateExpiryAlert),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
}

// check is called once before dialing, then only by the monitor's goroutine.
// Using the certificate fingerprint also deduplicates certificates that occur
// in both the client chain and CA bundle.
func (monitor *certificateExpiryMonitor) check(now time.Time) {
	for _, item := range monitor.certificates {
		cert := item.certificate
		remaining := cert.NotAfter.Sub(now)
		if remaining > certificateExpiryWarningWindow {
			continue
		}

		level, state := zap.WarnLevel, "expires soon"
		if remaining <= 0 {
			level, state = zap.ErrorLevel, "has expired"
		}
		fingerprint := sha256.Sum256(cert.Raw)
		previous, alerted := monitor.alerts[fingerprint]
		if alerted && previous.level == level && now.Sub(previous.time) < certificateExpiryRepeatInterval {
			continue
		}

		// Email and Telegram log hooks receive Entry.Message, but not structured
		// fields. Keep the certificate identity and expiry in the message itself.
		message := fmt.Sprintf("[etcd] %s certificate %s: path=%q, subject=%q, SHA256=%x, expires_at=%s, remaining=%s. Renew the certificate and recreate the etcd client (or restart the daemon) to load it.",
			item.role, state, item.path, cert.Subject.String(), fingerprint,
			cert.NotAfter.UTC().Format(time.RFC3339), remaining.Round(time.Second))
		if entry := monitor.logger.Check(level, message); entry != nil {
			entry.Write()
			monitor.alerts[fingerprint] = certificateExpiryAlert{level: level, time: now}
		}
	}
}

// start is called once, only after the etcd client has initialized successfully.
func (monitor *certificateExpiryMonitor) start() {
	go func() {
		ticker := time.NewTicker(certificateExpiryCheckInterval)
		defer ticker.Stop()
		defer close(monitor.done)
		for {
			select {
			case <-monitor.stop:
				return
			case now := <-ticker.C:
				monitor.check(now)
			}
		}
	}()
}

func (monitor *certificateExpiryMonitor) close() {
	monitor.stopOnce.Do(func() { close(monitor.stop) })
	<-monitor.done
}
