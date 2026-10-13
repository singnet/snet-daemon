package etcddb

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEtcdClientConnectsAfterServerStarts(t *testing.T) {
	servers, endpoints := newEtcdTestCluster(t, 1)
	metadata := newEtcdTestMetadata(t, endpoints, 500*time.Millisecond)
	started := time.Now()
	client, err := NewEtcdClientFromVip(nil, metadata)
	require.NoError(t, err, "an offline cluster must not prevent client initialization")
	t.Cleanup(client.Close)
	require.Less(t, time.Since(started), 500*time.Millisecond,
		"initialization must not wait for a storage request to time out")

	require.ErrorIs(t, client.Put("late-start", "unavailable"), context.DeadlineExceeded)
	value, present, err := client.Get("late-start")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, present)
	require.Empty(t, value)
	mutex, err := client.NewMutex("late-start-lock")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, mutex)

	require.NoError(t, servers[0].Start())
	require.Eventually(t, func() bool {
		return client.Put("late-start", "available") == nil
	}, 10*time.Second, 100*time.Millisecond, "the same client must connect when etcd becomes available")
	value, present, err = client.Get("late-start")
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "available", value)
	mutex, err = client.NewMutex("late-start-lock")
	require.NoError(t, err, "session creation must be retryable after an offline request")

	// The short timeout for granting the lease must not end the keepalive.
	select {
	case <-client.session.value.Done():
		t.Fatal("the session ended with its initial request deadline")
	case <-time.After(2 * client.timeout):
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, mutex.Lock(ctx))
	require.NoError(t, mutex.Unlock(ctx))
}

func TestEtcdClientCloseCancelsPendingSession(t *testing.T) {
	// Accept TCP connections without serving etcd so a lease request stays pending.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	metadata := newEtcdTestMetadata(t, []string{"http://" + listener.Addr().String()}, 10*time.Second)
	client, err := NewEtcdClientFromVip(nil, metadata)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	result := make(chan error, 1)
	go func() {
		_, err := client.NewMutex("pending-lock")
		result <- err
	}()
	require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(3*time.Second)))
	conn, err := listener.Accept()
	require.NoError(t, err, "the lease request must start connecting before closing the client")
	t.Cleanup(func() { _ = conn.Close() })

	started := time.Now()
	client.Close()
	require.Less(t, time.Since(started), time.Second, "close must cancel the pending lease request")
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("the pending mutex request did not stop after closing the client")
	}
	_, err = client.NewMutex("closed-client")
	require.ErrorIs(t, err, context.Canceled)
	client.Close()
}

func (suite *EtcdTestSuite) TestClientUsesAvailableEndpoint() {
	t := suite.T()
	endpoints := append([]string{"http://127.0.0.1:1"}, suite.metaData.GetPaymentStorageEndPoints()...)
	metadata := newEtcdTestMetadata(t, endpoints, 3*time.Second)
	client, err := NewEtcdClientFromVip(nil, metadata)
	require.NoError(t, err, "the first endpoint need not be available")
	defer client.Close()
	require.NoError(t, client.Put(t.Name(), "available endpoint"))
	value, present, err := client.Get(t.Name())
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "available endpoint", value)
}

func (suite *EtcdTestSuite) TestConcurrentMutexesShareOneSession() {
	t := suite.T()
	const count = 8
	errors := make([]error, count)
	var started sync.WaitGroup
	for i := range count {
		started.Add(1)
		go func() {
			defer started.Done()
			_, errors[i] = suite.client.NewMutex("shared-session")
		}()
	}
	started.Wait()
	for _, err := range errors {
		require.NoError(t, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	leases, err := suite.client.etcd.Leases(ctx)
	require.NoError(t, err)
	require.Len(t, leases.Leases, 1, "concurrent mutex creation must grant only one lease")
}
