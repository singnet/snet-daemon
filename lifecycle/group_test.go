package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCancelWaitsForWorkerCleanup(t *testing.T) {
	group := New()
	cleanup, release, stopped := make(chan struct{}), make(chan struct{}), make(chan struct{})
	require.True(t, group.Go(func(ctx context.Context) { <-ctx.Done(); close(cleanup); <-release }))
	group.Cancel()
	select {
	case <-cleanup:
	case <-time.After(time.Second):
		t.Fatal("worker did not receive cancellation")
	}
	go func() { group.Wait(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("wait returned before cleanup")
	default:
	}
	require.False(t, group.Go(func(context.Context) { t.Error("work admitted after cancellation") }))
	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("worker cleanup did not finish")
	}
}

func TestStopAcceptingPreservesActiveRequestUntilCancellation(t *testing.T) {
	group := New()
	ctx, done, ok := group.Begin(t.Context())
	require.True(t, ok)
	group.StopAccepting()
	require.NoError(t, ctx.Err())
	_, _, ok = group.Begin(t.Context())
	require.False(t, ok)
	group.Cancel()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("active request was not cancelled")
	}
	done()
	group.Wait()
}
