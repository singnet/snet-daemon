package metrics

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/lifecycle"
	"github.com/stretchr/testify/require"
)

func TestMeteringWorkerCancellationClosesPendingRequest(t *testing.T) {
	previous := config.Vip().Get(config.PvtKeyForMetering)
	config.Vip().Set(config.PvtKeyForMetering, "invalid")
	t.Cleanup(func() { config.Vip().Set(config.PvtKeyForMetering, previous) })
	entered, cancelled := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	}))
	t.Cleanup(server.Close)
	workers := lifecycle.New()
	t.Cleanup(func() { workers.Cancel(); workers.Wait() })
	require.True(t, workers.Go(func(ctx context.Context) {
		PublishContext(ctx, map[string]int{"value": 1}, server.URL, &CommonStats{}, big.NewInt(1))
	}))
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("metering request did not start")
	}
	workers.Cancel()
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("metering worker did not stop")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("metering transport was not cancelled")
	}
}
