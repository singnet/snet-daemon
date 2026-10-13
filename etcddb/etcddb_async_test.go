package etcddb

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Override only the lease operations a test controls; all other behavior stays
// with the real client, without package-level hooks or shared mutable fixtures.
type sessionTestLease struct {
	clientv3.Lease
	grant     func(context.Context, int64) (*clientv3.LeaseGrantResponse, error)
	keepAlive func(context.Context, clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error)
	revoke    func(context.Context, clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error)
}

func (lease *sessionTestLease) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	if lease.grant != nil {
		return lease.grant(ctx, ttl)
	}
	return lease.Lease.Grant(ctx, ttl)
}

func (lease *sessionTestLease) KeepAlive(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	if lease.keepAlive != nil {
		return lease.keepAlive(ctx, id)
	}
	return lease.Lease.KeepAlive(ctx, id)
}

func (lease *sessionTestLease) Revoke(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
	if lease.revoke != nil {
		return lease.revoke(ctx, id)
	}
	return lease.Lease.Revoke(ctx, id)
}

func newSessionTestClient(t *testing.T) *EtcdClient {
	t.Helper()
	metadata := newEtcdTestMetadata(t, []string{"http://127.0.0.1:1"}, 5*time.Second)
	client, err := NewEtcdClientFromVip(nil, metadata)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

func TestEtcdClientConcurrentSessionFailureSharesAttempt(t *testing.T) {
	client := newSessionTestClient(t)
	grantStarted := make(chan struct{})
	releaseGrant := make(chan struct{})
	var started sync.Once
	var grants atomic.Int64
	grantErr := errors.New("controlled lease grant failure")
	client.etcd.Lease = &sessionTestLease{
		Lease: client.etcd.Lease,
		grant: func(ctx context.Context, _ int64) (*clientv3.LeaseGrantResponse, error) {
			grants.Add(1)
			started.Do(func() { close(grantStarted) })
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-releaseGrant:
				return nil, grantErr
			}
		},
	}
	firstResult := make(chan error, 1)
	go func() {
		_, err := client.NewMutex("first-lock")
		firstResult <- err
	}()
	select {
	case <-grantStarted:
	case <-time.After(time.Second):
		t.Fatal("the first lease grant did not start")
	}

	// Joining callers have shorter deadlines than the initialization they
	// share. They must time out without waiting for a mutex held over Grant.
	shortClient := *client
	shortClient.timeout = 75 * time.Millisecond
	type result struct {
		mutex *EtcdClientMutex
		err   error
	}
	const count = 8
	results := make(chan result, count)
	start := make(chan struct{})
	for range count {
		go func() {
			<-start
			mutex, err := shortClient.NewMutex("joining-lock")
			results <- result{mutex: mutex, err: err}
		}()
	}
	close(start)
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for range count {
		select {
		case result := <-results:
			require.Nil(t, result.mutex)
			require.ErrorIs(t, result.err, context.DeadlineExceeded)
		case <-deadline.C:
			t.Fatal("joining callers waited for shared initialization instead of their own deadlines")
		}
	}
	require.Equal(t, int64(1), grants.Load(), "joining callers must not start additional lease grants")
	close(releaseGrant)
	select {
	case err := <-firstResult:
		require.ErrorIs(t, err, grantErr)
	case <-time.After(time.Second):
		t.Fatal("the initial caller did not receive the shared attempt's failure")
	}
	_, err := client.NewMutex("retry-lock")
	require.ErrorIs(t, err, grantErr)
	require.Equal(t, int64(2), grants.Load(), "a finished failure must not prevent a new attempt")
}

func TestEtcdClientSessionCleanupFailurePreservesError(t *testing.T) {
	for _, cleanupTimesOut := range []bool{false, true} {
		name := "revoke error"
		if cleanupTimesOut {
			name = "revoke timeout"
		}
		t.Run(name, func(t *testing.T) {
			client := newSessionTestClient(t)
			sessionErr := errors.New("controlled keepalive failure")
			cleanupErr := errors.New("controlled revoke failure")
			const grantedID clientv3.LeaseID = 42
			var grantCtx context.Context
			var revokedID clientv3.LeaseID
			var cleanupCtxErr error
			var cleanupRemaining time.Duration
			var cleanupHasDeadline bool
			client.etcd.Lease = &sessionTestLease{
				Lease: client.etcd.Lease,
				grant: func(ctx context.Context, _ int64) (*clientv3.LeaseGrantResponse, error) {
					grantCtx = ctx
					return &clientv3.LeaseGrantResponse{ID: grantedID}, nil
				},
				keepAlive: func(context.Context, clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
					return nil, sessionErr
				},
				revoke: func(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
					revokedID, cleanupCtxErr = id, ctx.Err()
					deadline, ok := ctx.Deadline()
					cleanupHasDeadline, cleanupRemaining = ok, time.Until(deadline)
					if cleanupTimesOut {
						<-ctx.Done()
						return nil, ctx.Err()
					}
					return nil, cleanupErr
				},
			}
			mutex, err := client.NewMutex("failed-session-lock")
			require.Nil(t, mutex)
			require.ErrorIs(t, err, sessionErr)
			require.NotErrorIs(t, err, cleanupErr)
			require.NotErrorIs(t, err, context.DeadlineExceeded)
			require.Equal(t, grantedID, revokedID)
			require.ErrorIs(t, grantCtx.Err(), context.Canceled)
			require.NoError(t, cleanupCtxErr, "cleanup must not reuse the canceled grant context")
			require.True(t, cleanupHasDeadline)
			require.Positive(t, cleanupRemaining)
			require.LessOrEqual(t, cleanupRemaining, etcdLeaseCleanupTimeout)
		})
	}
}

func TestEtcdClientCloseDuringSessionCreation(t *testing.T) {
	client := newSessionTestClient(t)
	keepAliveStarted := make(chan struct{})
	releaseKeepAlive := make(chan struct{})
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(releaseKeepAlive) }) })
	const grantedID clientv3.LeaseID = 43
	var revokedID clientv3.LeaseID
	var revocations atomic.Int64
	var transportErrDuringCleanup error
	var cleanupCtxErr error
	client.etcd.Lease = &sessionTestLease{
		Lease: client.etcd.Lease,
		grant: func(context.Context, int64) (*clientv3.LeaseGrantResponse, error) {
			return &clientv3.LeaseGrantResponse{ID: grantedID}, nil
		},
		keepAlive: func(ctx context.Context, _ clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
			close(keepAliveStarted)
			<-releaseKeepAlive
			responses := make(chan *clientv3.LeaseKeepAliveResponse)
			go func() {
				<-ctx.Done()
				close(responses)
			}()
			return responses, nil
		},
		revoke: func(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
			revocations.Add(1)
			revokedID = id
			transportErrDuringCleanup, cleanupCtxErr = client.etcd.Ctx().Err(), ctx.Err()
			return &clientv3.LeaseRevokeResponse{}, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		_, err := client.NewMutex("closing-session-lock")
		result <- err
	}()
	select {
	case <-keepAliveStarted:
	case <-time.After(time.Second):
		t.Fatal("session creation did not reach keepalive initialization")
	}
	closed := make(chan struct{}, 2)
	for range 2 {
		go func() {
			client.Close()
			closed <- struct{}{}
		}()
	}
	select {
	case <-client.session.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("close did not cancel session initialization")
	}
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("the mutex caller did not stop while initialization was pending")
	}
	_, err := client.NewMutex("already-closing-lock")
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-closed:
		t.Fatal("close must finish unpublished-session cleanup before closing the transport")
	default:
	}
	release.Do(func() { close(releaseKeepAlive) })
	for range 2 {
		select {
		case <-closed:
		case <-time.After(time.Second):
			t.Fatal("concurrent close did not finish after session initialization was released")
		}
	}
	require.Equal(t, grantedID, revokedID)
	require.Equal(t, int64(1), revocations.Load(), "concurrent shutdown must clean up the lease only once")
	require.NoError(t, transportErrDuringCleanup)
	require.NoError(t, cleanupCtxErr)
	require.Nil(t, client.session.value, "a session must not be published after shutdown starts")
	client.Close()
}

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

func (suite *EtcdTestSuite) TestSessionFailureRevokesLease() {
	t := suite.T()
	client := suite.client
	base := client.etcd.Lease
	sessionErr := errors.New("controlled keepalive failure")
	var failKeepAlive atomic.Bool
	failKeepAlive.Store(true)
	var grantedID atomic.Int64
	revokedIDs := make(chan clientv3.LeaseID, 2)
	client.etcd.Lease = &sessionTestLease{
		Lease: base,
		grant: func(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
			lease, err := base.Grant(ctx, ttl)
			if err == nil {
				grantedID.Store(int64(lease.ID))
			}
			return lease, err
		},
		keepAlive: func(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
			if failKeepAlive.CompareAndSwap(true, false) {
				return nil, sessionErr
			}
			return base.KeepAlive(ctx, id)
		},
		revoke: func(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
			revokedIDs <- id
			return base.Revoke(ctx, id)
		},
	}
	mutex, err := client.NewMutex("failed-keepalive-lock")
	require.Nil(t, mutex)
	require.ErrorIs(t, err, sessionErr)
	select {
	case id := <-revokedIDs:
		require.Equal(t, clientv3.LeaseID(grantedID.Load()), id)
	case <-time.After(time.Second):
		t.Fatal("the failed session's lease was not revoked")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	leases, err := client.etcd.Leases(ctx)
	require.NoError(t, err)
	require.Empty(t, leases.Leases, "the failed session must not leave a lease until TTL expiry")
	mutex, err = client.NewMutex("recovered-keepalive-lock")
	require.NoError(t, err, "session creation must remain retryable after keepalive fails")
	require.NotNil(t, mutex)
	leases, err = client.etcd.Leases(ctx)
	require.NoError(t, err)
	require.Len(t, leases.Leases, 1)
}
