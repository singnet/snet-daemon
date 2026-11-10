package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/etcddb"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type delayedShutdownUnlock struct {
	storage.AtomicStorage
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *delayedShutdownUnlock) CompareAndSwap(key, previous, next string) (bool, error) {
	if previous == "locked" && next == "unlocked" {
		s.once.Do(func() { close(s.started) })
		<-s.release
	}
	return s.AtomicStorage.CompareAndSwap(key, previous, next)
}

func TestDaemonShutdownDeadlineWaitsForPaymentCleanup(t *testing.T) {
	endpoint := shutdownEtcd(t)
	backend, entered, unblock := shutdownBackend(t, "success")
	components := shutdownEtcdComponents(t, backend, endpoint, t.Name())
	config.Vip().Set(config.ShutdownTimeout, "50ms")
	gate := &delayedShutdownUnlock{AtomicStorage: components.atomicStorage, started: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(gate.release) }) }
	t.Cleanup(release)
	components.atomicStorage = gate
	seedShutdownChannel(t, components)
	observerClient, err := etcddb.NewEtcdClientFromVip(config.Vip(), components.OrganizationMetaData())
	require.NoError(t, err)
	observer := &Components{serviceMetadata: components.serviceMetadata, organizationMetaData: components.organizationMetaData,
		atomicStorage: storage.NewPrefixedAtomicStorage(observerClient, t.Name()), etcdClient: observerClient}
	t.Cleanup(observer.Close)
	d, err := newDaemon(components)
	require.NoError(t, err)
	d.start()
	t.Cleanup(func() { release(); unblock(); d.stop() })
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	client := shutdownClient(t, d.lis.Addr().String())
	callDone := make(chan error, 1)
	payment := shutdownPaymentContext(t, ctx, 1)
	go func() { callDone <- shutdownCall(payment, client) }()
	waitShutdown(t, entered, "the paid call")
	stopped := make(chan struct{})
	go func() { d.stop(); components.Close(); close(stopped) }()
	waitShutdown(t, gate.started, "rollback after the shutdown deadline")
	select {
	case <-stopped:
		t.Fatal("shutdown returned before channel unlock finished")
	default:
	}
	_, _, err = components.etcdClient.Get("cleanup-can-still-use-storage")
	require.NoError(t, err, "etcd must remain open during payment cleanup")
	release()
	waitShutdown(t, stopped, "shutdown after payment cleanup")
	require.Error(t, <-callDone)
	assertShutdownChannel(t, observer, false, 0)
}

func assertShutdownPortReleased(t *testing.T, address string) {
	t.Helper()
	listener, err := net.Listen("tcp", address)
	require.NoError(t, err, "shutdown must release %s", address)
	require.NoError(t, listener.Close())
}

func TestDaemonShutdownHTTP(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%t", deadline), func(t *testing.T) {
			conf := lifecycleConfig(t)
			conf.Set(config.DaemonTypeKey, "http")
			conf.Set(config.PassthroughEnabledKey, true)
			if deadline {
				conf.Set(config.ShutdownTimeout, "50ms")
			}
			entered, release, cancelled := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				close(entered)
				select {
				case <-release:
					_, _ = io.WriteString(w, "finished")
				case <-r.Context().Done():
					close(cancelled)
				}
			}))
			t.Cleanup(func() { unblock(); backend.Close() })
			conf.Set(config.ServiceEndpointKey, backend.URL)
			components := lifecycleComponents(t)
			d, err := newDaemon(components)
			require.NoError(t, err)
			address := d.lis.Addr().String()
			listener := &shutdownListener{Listener: d.lis, closed: make(chan struct{})}
			d.lis = listener
			d.start()
			t.Cleanup(func() { unblock(); d.stop() })
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+address, strings.NewReader("request"))
			require.NoError(t, err)
			done := make(chan error, 1)
			go func() {
				response, err := http.DefaultClient.Do(request)
				if err == nil {
					defer response.Body.Close()
					var body []byte
					body, err = io.ReadAll(response.Body)
					if !deadline && string(body) != "finished" {
						err = fmt.Errorf("unexpected response %q", body)
					}
				}
				done <- err
			}()
			waitShutdown(t, entered, "HTTP backend request")
			stopped := make(chan struct{})
			go func() { d.stop(); close(stopped) }()
			waitShutdown(t, listener.closed, "HTTP listener shutdown")
			if deadline {
				waitShutdown(t, cancelled, "outgoing HTTP request cancellation")
			} else {
				select {
				case <-stopped:
					t.Fatal("shutdown did not wait for HTTP request")
				default:
				}
				unblock()
			}
			waitShutdown(t, stopped, "HTTP server shutdown")
			if err := <-done; !deadline {
				require.NoError(t, err)
			}
			assertShutdownPortReleased(t, address)
			d.stop()
		})
	}
}

func shutdownWebCall(t *testing.T, client *http.Client, address string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	md, _ := metadata.FromOutgoingContext(shutdownPaymentContext(t, ctx, 1))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address+shutdownMethod, bytes.NewReader(make([]byte, 5)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/grpc-web+proto")
	request.Header.Set("X-Grpc-Web", "1")
	for key, values := range md {
		for _, value := range values {
			if strings.HasSuffix(key, "-bin") {
				value = base64.StdEncoding.EncodeToString([]byte(value))
			}
			request.Header.Add(key, value)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("grpc-status: 0")) {
		return fmt.Errorf("gRPC-Web response: status %d, body %q", response.StatusCode, body)
	}
	return nil
}

func TestDaemonShutdownGRPCWeb(t *testing.T) {
	for _, mode := range []string{"fork", "original", "split_web", "traffic_split", "tls"} {
		for _, deadline := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/deadline=%t", mode, deadline), func(t *testing.T) {
				backend, entered, unblock := shutdownBackend(t, "success")
				components := shutdownComponents(t, backend, storage.NewMemStorage())
				experimental := map[string]any{"use_original_cmux": mode == "original", "split_grpc_web": mode == "split_web"}
				if mode == "traffic_split" {
					experimental["traffic_split"] = map[string]any{"grpc": 0, "http": 0}
				}
				config.Vip().Set(config.Experimental, experimental)
				if deadline {
					config.Vip().Set(config.ShutdownTimeout, "100ms")
				}
				transport := &http.Transport{}
				t.Cleanup(transport.CloseIdleConnections)
				if mode == "tls" {
					cert, key, cleanup := generateSelfSignedCert(t)
					t.Cleanup(cleanup)
					config.Vip().Set(config.SSLCertPathKey, cert)
					config.Vip().Set(config.SSLKeyPathKey, key)
					transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // Local self-signed test certificate.
				}
				seedShutdownChannel(t, components)
				d, err := newDaemon(components)
				require.NoError(t, err)
				d.start()
				t.Cleanup(func() { unblock(); d.stop() })
				address := d.listeners[0].Addr().String()
				if mode == "traffic_split" {
					address = d.listeners[1].Addr().String()
				}
				scheme := "http://"
				if mode == "tls" {
					scheme = "https://"
				}
				result := make(chan error, 1)
				go func() { result <- shutdownWebCall(t, &http.Client{Transport: transport}, scheme+address) }()
				waitShutdown(t, entered, "paid gRPC-Web backend call")
				assertShutdownChannel(t, components, true, 0)
				stopped := make(chan struct{})
				go func() { d.stop(); close(stopped) }()
				if deadline {
					waitShutdown(t, stopped, "gRPC-Web deadline cancellation")
					require.Error(t, <-result)
					assertShutdownChannel(t, components, false, 0)
				} else {
					require.Eventually(t, d.stopping.Load, time.Second, time.Millisecond)
					assertShutdownChannel(t, components, true, 0)
					unblock()
					require.NoError(t, <-result)
					waitShutdown(t, stopped, "graceful gRPC-Web completion")
					assertShutdownChannel(t, components, false, 1)
				}
				for _, listener := range d.listeners {
					assertShutdownPortReleased(t, listener.Addr().String())
				}
			})
		}
	}
}

func TestDaemonShutdownClosesUnclassifiedConnections(t *testing.T) {
	lifecycleConfig(t)
	d, err := newDaemon(lifecycleComponents(t))
	require.NoError(t, err)
	d.start()
	t.Cleanup(d.stop)
	conn, err := net.DialTimeout("tcp", d.lis.Addr().String(), time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.Eventually(t, func() bool {
		d.connections.mu.Lock()
		defer d.connections.mu.Unlock()
		return len(d.connections.conns) != 0
	}, time.Second, time.Millisecond)
	stopped := make(chan struct{})
	go func() { d.stop(); close(stopped) }()
	waitShutdown(t, stopped, "shutdown with an unfinished cmux handshake")
}

func TestDaemonShutdownClosesACMEAndStopsWorkers(t *testing.T) {
	conf := lifecycleConfig(t)
	conf.Set(config.DaemonTypeKey, "http")
	components := lifecycleComponents(t)
	workerStarted, workerStopped := make(chan struct{}), make(chan struct{})
	components.Background().Go(func(ctx context.Context) { close(workerStarted); <-ctx.Done(); close(workerStopped) })
	mainListener, acmeListener := testListener(t), testListener(t)
	d := &daemon{lis: mainListener, acmeListener: acmeListener, autoSSLDomain: "example.com", components: components}
	d.start()
	t.Cleanup(d.stop)
	waitShutdown(t, workerStarted, "owned background worker")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get("http://" + acmeListener.Addr().String() + "/.well-known/acme-challenge/missing")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	var cleanupCalls atomic.Int32
	proxyCleanup := d.closeProxy
	d.closeProxy = func() { cleanupCalls.Add(1); proxyCleanup() }
	var stops sync.WaitGroup
	for range 4 {
		stops.Add(1)
		go func() { defer stops.Done(); d.stop() }()
	}
	stops.Wait()
	waitShutdown(t, workerStopped, "background worker cancellation")
	require.Equal(t, int32(1), cleanupCalls.Load())
	assertShutdownPortReleased(t, mainListener.Addr().String())
	assertShutdownPortReleased(t, acmeListener.Addr().String())
}
