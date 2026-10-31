package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/etcddb"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/server/v3/embed"
)

func shutdownEtcd(t *testing.T) string {
	t.Helper()
	conf := embed.NewConfig()
	conf.Dir = t.TempDir()
	conf.LogLevel, conf.LogOutputs = "error", []string{"stderr"}
	listenURL := url.URL{Scheme: "http", Host: "127.0.0.1:0"}
	conf.ListenClientUrls, conf.AdvertiseClientUrls = []url.URL{listenURL}, []url.URL{listenURL}
	conf.ListenPeerUrls, conf.AdvertisePeerUrls = []url.URL{listenURL}, []url.URL{listenURL}
	conf.InitialCluster = conf.InitialClusterFromName(conf.Name)
	server, err := embed.StartEtcd(conf)
	require.NoError(t, err)
	t.Cleanup(server.Close)
	waitShutdown(t, server.Server.ReadyNotify(), "embedded etcd to start")
	return "http://" + server.Clients[0].Addr().String()
}

func shutdownEtcdComponents(t *testing.T, backend, endpoint, namespace string) *Components {
	t.Helper()
	components := shutdownComponents(t, backend, nil)
	conf := config.Vip()
	conf.Set(config.PaymentChannelStorageClientKey+".endpoints", []string{endpoint})
	conf.Set(config.PaymentChannelStorageClientKey+".connection_timeout", "3s")
	conf.Set(config.PaymentChannelStorageClientKey+".request_timeout", "3s")
	conf.Set(config.PaymentChannelStorageClientKey+".hot_reload", false)
	client, err := etcddb.NewEtcdClientFromVip(conf, components.OrganizationMetaData())
	require.NoError(t, err)
	components.etcdClient = client
	components.atomicStorage = storage.NewPrefixedAtomicStorage(client, namespace)
	return components
}

type shutdownChild struct {
	command *exec.Cmd
	lines   chan string
	done    chan struct{}
	err     error
	stderr  bytes.Buffer
	stdout  strings.Builder
}

func startShutdownChild(t *testing.T, backend, endpoint, namespace string, echo bool) (*shutdownChild, string) {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, executable, "-test.run=^TestDaemonShutdownSignalHelper$", "-test.timeout=25s")
	repoRoot, err := os.Getwd()
	require.NoError(t, err)
	command.Dir = filepath.Join(repoRoot, "snetd", "cmd")
	command.Env = append(os.Environ(), "SNET_SHUTDOWN_HELPER=1",
		"SNET_SHUTDOWN_TIMEOUT="+config.ShutdownGracePeriod().String(),
		"SNET_SHUTDOWN_BACKEND="+backend, "SNET_SHUTDOWN_ETCD="+endpoint,
		"SNET_SHUTDOWN_NAMESPACE="+namespace, fmt.Sprintf("SNET_SHUTDOWN_ECHO=%t", echo))
	child := &shutdownChild{command: command, lines: make(chan string, 4), done: make(chan struct{})}
	output, err := command.StdoutPipe()
	require.NoError(t, err)
	command.Stderr = &child.stderr
	require.NoError(t, command.Start())
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			line := scanner.Text()
			child.stdout.WriteString(line + "\n")
			if strings.HasPrefix(line, "SHUTDOWN_") {
				child.lines <- line
			}
		}
		child.err = errors.Join(scanner.Err(), command.Wait())
		close(child.done)
	}()
	t.Cleanup(func() {
		select {
		case <-child.done:
		default:
			_ = command.Process.Kill()
			waitShutdown(t, child.done, "test child cleanup")
		}
	})
	address := strings.TrimPrefix(child.waitLine(t, "SHUTDOWN_READY "), "SHUTDOWN_READY ")
	return child, address
}

func (child *shutdownChild) waitLine(t *testing.T, prefix string) string {
	t.Helper()
	select {
	case line := <-child.lines:
		require.True(t, strings.HasPrefix(line, prefix), "unexpected child message: %s", line)
		return line
	case <-child.done:
		t.Fatalf("child exited unexpectedly: %v\n%s%s", child.err, child.stdout.String(), child.stderr.String())
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for child message %q", prefix)
	}
	return ""
}

func (child *shutdownChild) wait(t *testing.T) {
	t.Helper()
	waitShutdown(t, child.done, "daemon process to exit")
	require.NoError(t, child.err, "daemon output:\n%s%s", child.stdout.String(), child.stderr.String())
}

// Run the production signal/stop path with real payment storage and mocked Ethereum.
// The parent owns etcd, so exiting the daemon cannot erase a leaked channel lock.
func TestDaemonShutdownSignalHelper(t *testing.T) {
	if os.Getenv("SNET_SHUTDOWN_HELPER") != "1" {
		return
	}
	components := shutdownEtcdComponents(t, os.Getenv("SNET_SHUTDOWN_BACKEND"),
		os.Getenv("SNET_SHUTDOWN_ETCD"), os.Getenv("SNET_SHUTDOWN_NAMESPACE"))
	config.Vip().Set(config.ShutdownTimeout, os.Getenv("SNET_SHUTDOWN_TIMEOUT"))
	config.Vip().Set(config.PassthroughEnabledKey, os.Getenv("SNET_SHUTDOWN_ECHO") != "true")
	d, err := newDaemon(components)
	require.NoError(t, err)
	d.lis = &shutdownListener{Listener: d.lis, closed: make(chan struct{}), onClose: func() {
		fmt.Println("SHUTDOWN_DRAINING")
	}}
	fmt.Println("SHUTDOWN_READY " + d.lis.Addr().String())
	d.serve()
}

func TestDaemonShutdownSignals(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("SIGINT/SIGTERM subprocess coverage runs on Linux CI")
	}
	endpoint := shutdownEtcd(t)
	for _, signal := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		for _, outcome := range []string{"success", "error", "cancel", "timeout"} {
			t.Run(fmt.Sprintf("%s/%s", signal, outcome), func(t *testing.T) {
				backend, entered, unblock := shutdownBackend(t, outcome)
				components := shutdownEtcdComponents(t, backend, endpoint, t.Name())
				if outcome == "timeout" {
					config.Vip().Set(config.ShutdownTimeout, "200ms")
				}
				seedShutdownChannel(t, components)
				child, address := startShutdownChild(t, backend, endpoint, t.Name(), false)
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				callDone := make(chan error, 1)
				client := shutdownClient(t, address)
				paymentCtx := shutdownPaymentContext(t, ctx, 1)
				go func() { callDone <- shutdownCall(paymentCtx, client) }()
				waitShutdown(t, entered, "the child's paid backend call")
				assertShutdownChannel(t, components, true, 0)
				require.NoError(t, child.command.Process.Signal(signal))
				child.waitLine(t, "SHUTDOWN_DRAINING")
				select {
				case <-child.done:
					t.Fatal("daemon exited before its active payment completed")
				default:
				}
				assertShutdownChannel(t, components, true, 0)
				if outcome == "cancel" {
					cancel()
				} else if outcome != "timeout" {
					unblock()
				}
				child.wait(t)
				amount := shutdownAmount(t, outcome, <-callDone)
				assertShutdownChannel(t, components, false, amount)

				// Do not seed state in the child: the restarted process must read
				// the balance and unlocked channel left by the previous process.
				restarted, address := startShutdownChild(t, backend, endpoint, t.Name(), true)
				assertShutdownChannel(t, components, false, amount)
				retryCtx, retryCancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer retryCancel()
				require.NoError(t, shutdownCall(shutdownPaymentContext(t, retryCtx, amount+1), shutdownClient(t, address)))
				require.NoError(t, restarted.command.Process.Signal(signal))
				restarted.wait(t)
				assertShutdownChannel(t, components, false, amount+1)
			})
		}
	}
}
