package training

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type acceptNotifyingListener struct {
	net.Listener
	accepting chan struct{}
}

func (l *acceptNotifyingListener) Accept() (net.Conn, error) {
	select {
	case l.accepting <- struct{}{}:
	default:
	}
	return l.Listener.Accept()
}

func TestStopCoverageProvider(t *testing.T) {
	for _, tc := range []struct {
		name            string
		stopBeforeServe bool
	}{
		{name: "stop before serve", stopBeforeServe: true},
		{name: "stop after serve"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			server := grpc.NewServer()
			t.Cleanup(server.Stop)
			if tc.stopBeforeServe {
				server.Stop()
			}

			ready := &acceptNotifyingListener{Listener: listener, accepting: make(chan struct{}, 1)}
			done := make(chan error, 1)
			go func() { done <- server.Serve(ready) }()
			if !tc.stopBeforeServe {
				select {
				case <-ready.accepting:
				case <-time.After(5 * time.Second):
					t.Fatal("provider did not start accepting connections")
				}
			}

			require.NoError(t, stopCoverageProvider(server, done))
		})
	}
}

func TestStopCoverageProviderPreservesServeError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	done := make(chan error, 1)
	done <- server.Serve(listener)

	require.ErrorIs(t, stopCoverageProvider(server, done), net.ErrClosed)
}
