package cmd

import (
	"net"
	"testing"

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
