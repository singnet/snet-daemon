package escrow

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
)

// isolateConfig replaces only Viper, not other config package state.
// Tests using it must not run in parallel and must finish background config
// accesses before cleanup restores the previous instance.
func isolateConfig(t *testing.T) {
	t.Helper()
	previous := config.Vip()
	config.SetVip(viper.New())
	t.Cleanup(func() { config.SetVip(previous) })
}

// currentBlockForStatsTest observes the asynchronous publisher without making
// blockchain or metering requests.
func currentBlockForStatsTest(t *testing.T) func() (*big.Int, error) {
	t.Helper()
	isolateConfig(t)
	config.Vip().Set(config.MeteringEnabled, true)
	called := make(chan struct{})
	// Registered after isolateConfig so this runs before Viper is restored.
	t.Cleanup(func() {
		select {
		case <-called:
		case <-time.After(5 * time.Second):
			t.Error("Complete did not invoke the stats publisher")
		}
	})
	return func() (*big.Int, error) {
		// PublishChannelStats has read its config by this point. Returning an
		// error prevents further config reads and HTTP requests in metrics.Publish.
		close(called)
		return nil, errors.New("test block unavailable")
	}
}
