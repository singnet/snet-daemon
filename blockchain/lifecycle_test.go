package blockchain

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
)

func TestProcessorCancelsPendingRPC(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "shutdown", true: "timeout"}[deadline], func(t *testing.T) {
			previous := config.Vip().Get(config.ServiceTimeout)
			config.Vip().Set(config.ServiceTimeout, "50ms")
			t.Cleanup(func() { config.Vip().Set(config.ServiceTimeout, previous) })
			entered, cancelled := make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				close(entered)
				<-r.Context().Done()
				close(cancelled)
			}))
			t.Cleanup(server.Close)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			client, err := rpc.DialContext(ctx, server.URL)
			require.NoError(t, err)
			t.Cleanup(client.Close)
			p := &processor{ctx: ctx, ethHttpClient: ethclient.NewClient(client)}
			done := make(chan error, 1)
			go func() { _, err := p.CurrentBlock(); done <- err }()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("RPC did not start")
			}
			if !deadline {
				cancel()
			}
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("RPC ignored cancellation")
			}
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("RPC transport was not cancelled")
			}
		})
	}
}
