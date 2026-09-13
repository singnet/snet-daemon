package config

import (
	"net/http"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// Package state is process-wide: these tests must not use t.Parallel.
func isolatedConfig(t *testing.T) *viper.Viper {
	t.Helper()
	oldVip, oldNetwork := vip, networkSelected
	oldMapping, oldUsers := networkIdNameMapping, userAddress
	t.Cleanup(func() {
		SetVip(oldVip)
		networkSelected, networkIdNameMapping, userAddress = oldNetwork, oldMapping, oldUsers
	})
	v := viper.New()
	defaults := viper.New()
	require.NoError(t, ReadConfigFromJsonString(defaults, defaultConfigJson))
	SetDefaultFromConfig(v, defaults)
	SetVip(v)
	networkSelected, networkIdNameMapping, userAddress = &NetworkSelected{}, "", nil
	return v
}

type versionRoundTripper func(*http.Request) (*http.Response, error)

func (f versionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func stubVersionHTTP(t *testing.T, handler versionRoundTripper) {
	t.Helper()
	oldClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = oldClient })
	http.DefaultClient = &http.Client{Transport: handler}
}
