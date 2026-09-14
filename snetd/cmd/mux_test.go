package cmd

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func TestForkMuxEndpoints(t *testing.T) {
	t.Run("grpc and http endpoints without web split", func(t *testing.T) {
		m := newForkMux(testListener(t), false)
		eps := m.Endpoints()
		require.Len(t, eps, 2)
		assert.Equal(t, L_GRPC, eps[0].Type)
		assert.NotNil(t, eps[0].L)
		assert.Equal(t, L_HTTP, eps[1].Type)
		assert.NotNil(t, eps[1].L)
		assert.True(t, eps[0].L != eps[1].L)
	})

	t.Run("grpc web endpoint enabled with splitWeb", func(t *testing.T) {
		m := newForkMux(testListener(t), true)
		eps := m.Endpoints()
		require.Len(t, eps, 3)
		assert.Equal(t, L_GRPC, eps[0].Type)
		assert.Equal(t, L_GRPC_WEB, eps[1].Type)
		assert.Equal(t, L_HTTP, eps[2].Type)
	})
}

func TestOriginalMuxEndpoints(t *testing.T) {
	t.Run("grpc and http endpoints without web split", func(t *testing.T) {
		m := newOriginalMux(testListener(t), false)
		eps := m.Endpoints()
		require.Len(t, eps, 2)
		assert.Equal(t, L_GRPC, eps[0].Type)
		assert.Equal(t, L_HTTP, eps[1].Type)
	})

	t.Run("grpc web endpoint enabled with splitWeb", func(t *testing.T) {
		m := newOriginalMux(testListener(t), true)
		eps := m.Endpoints()
		require.Len(t, eps, 3)
		assert.Equal(t, L_GRPC, eps[0].Type)
		assert.Equal(t, L_GRPC_WEB, eps[1].Type)
		assert.Equal(t, L_HTTP, eps[2].Type)
	})
}

func TestMuxServeStopsAfterListenerClose(t *testing.T) {
	tests := []struct {
		name string
		new  func(net.Listener) GRPCMux
	}{
		{name: "fork", new: func(listener net.Listener) GRPCMux { return newForkMux(listener, false) }},
		{name: "original", new: func(listener net.Listener) GRPCMux { return newOriginalMux(listener, false) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener := testListener(t)
			mux := tt.new(listener)
			done := make(chan error, 1)
			go func() { done <- mux.Serve() }()

			require.NoError(t, listener.Close())
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("mux.Serve did not return after its listener was closed")
			}
		})
	}
}
