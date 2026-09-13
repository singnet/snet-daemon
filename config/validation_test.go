package config

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestValidateConfiguration(t *testing.T) {
	const address = "0x06A1D29e9FfA2415434A7A571235744F8DA2a514"
	for _, tc := range []struct {
		name     string
		settings map[string]any
		wantErr  string
	}{
		{"defaults", nil, ""},
		{"http daemon", map[string]any{DaemonTypeKey: "http"}, ""},
		{"unsupported daemon", map[string]any{DaemonTypeKey: "unknown"}, "unrecognized DAEMON_TYPE"},
		{"certificate without key", map[string]any{SSLCertPathKey: "cert.pem"}, "SSL requires both"},
		{"key without certificate", map[string]any{SSLKeyPathKey: "key.pem"}, "SSL requires both"},
		{"certificate and key", map[string]any{SSLCertPathKey: "cert.pem", SSLKeyPathKey: "key.pem"}, ""},
		{"endpoint loop", map[string]any{ServiceEndpointKey: "http://127.0.0.1:8080"}, "can't be the same"},
		{"missing daemon port", map[string]any{DaemonEndpoint: "localhost"}, "couldn't split"},
		{"invalid service URL", map[string]any{ServiceEndpointKey: "http://%zz"}, "valid url"},
		{"zero message size", map[string]any{MaxMessageSizeInMB: 0}, "max_message_size_in_mb"},
		{"negative message size", map[string]any{MaxMessageSizeInMB: -1}, "max_message_size_in_mb"},
		{"maximum message size", map[string]any{MaxMessageSizeInMB: 2048}, ""},
		{"oversized message", map[string]any{MaxMessageSizeInMB: 2049}, "max_message_size_in_mb"},
		{"restricted without users", map[string]any{AllowedUserFlag: true}, "valid Address needs to be specified"},
		{"restricted test network", map[string]any{AllowedUserFlag: true, AllowedUserAddresses: []string{address}}, ""},
		{"restricted mainnet", map[string]any{AllowedUserFlag: true, BlockChainNetworkSelected: "main"}, "service cannot be restricted"},
		{"invalid free call key", map[string]any{PvtKeyForFreeCalls: "invalid"}, "invalid private_key_for_free_calls"},
		{"valid free call key", map[string]any{PvtKeyForFreeCalls: strings.Repeat("0", 63) + "1"}, ""},
		{"short token secret", map[string]any{TokenSecretKey: strings.Repeat("x", 31)}, "at least 32 bytes"},
		{"minimum token secret", map[string]any{TokenSecretKey: strings.Repeat("x", 32)}, ""},
		{"disabled blockchain without secret", map[string]any{BlockchainEnabledKey: false, TokenSecretKey: ""}, ""},
		{"invalid metering URL", map[string]any{MeteringEnabled: true, MeteringEndpoint: "invalid"}, "valid Metering End point"},
		{"valid metering URL", map[string]any{MeteringEnabled: true, MeteringEndpoint: "https://metering.example.test"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := isolatedConfig(t)
			networkIdNameMapping = defaultBlockChainNetworkConfig
			stubVersionHTTP(t, func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"` + versionTag + `"}`))}, nil
			})
			for key, value := range tc.settings {
				v.Set(key, value)
			}
			err := Validate()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateNetworkConfigurationError(t *testing.T) {
	isolatedConfig(t)
	networkIdNameMapping = "invalid JSON"
	stubVersionHTTP(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("version request must not run after invalid network configuration")
		return nil, nil
	})
	require.Error(t, Validate())
}

func TestValidateToleratesVersionLookupFailure(t *testing.T) {
	isolatedConfig(t)
	networkIdNameMapping = defaultBlockChainNetworkConfig
	calls := 0
	stubVersionHTTP(t, func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("offline")
	})
	require.NoError(t, Validate(), "version availability must not prevent daemon startup")
	require.Equal(t, 1, calls)
}

func TestDeprecatedConfigurationAliases(t *testing.T) {
	for oldKey, newKey := range map[string]string{
		"daemon_end_point": DaemonEndpoint, "ipfs_end_point": IpfsEndpoint,
		"passthrough_endpoint": ServiceEndpointKey, "metering_end_point": MeteringEndpoint,
		"heartbeat_svc_end_point": HeartbeatServiceEndpoint, "notification_svc_end_point": NotificationServiceEndpoint,
		"pvt_key_for_metering": PvtKeyForMetering, "pvt_key_for_free_calls": PvtKeyForFreeCalls,
	} {
		t.Run(oldKey, func(t *testing.T) {
			v := viper.New()
			v.Set(oldKey, "legacy-value")
			v.Set("unrelated", "preserved")
			migrateDeprecatedParams(v)
			require.Equal(t, "legacy-value", v.GetString(newKey))
			require.Equal(t, "preserved", v.GetString("unrelated"))
			migrateDeprecatedParams(v)
			require.Equal(t, "legacy-value", v.GetString(newKey))
		})
	}
}

func TestDurationFallback(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  time.Duration
	}{
		{"valid", "2m30s", 150 * time.Second},
		{"fractional", "1.5s", 1500 * time.Millisecond},
		{"missing", nil, 100 * time.Second},
		{"numeric", 5, 100 * time.Second},
		{"invalid", "five seconds", 100 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := isolatedConfig(t)
			if tc.value != nil {
				v.Set("duration-under-test", tc.value)
			}
			require.Equal(t, tc.want, mustDuration("duration-under-test", 100*time.Second))
		})
	}
}
