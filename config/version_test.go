package config

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getVersionTag(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"", versionTag},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVersionTag(); got != tt.want {
				t.Errorf("getVersionTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSha1Revision(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"", sha1Revision},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSha1Revision(); got != tt.want {
				t.Errorf("getSha1Revision() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBuildTime(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"", buildTime},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBuildTime(); got != tt.want {
				t.Errorf("getBuildTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckVersionOfDaemon(t *testing.T) {
	oldTag := versionTag
	t.Cleanup(func() { versionTag = oldTag })
	stubVersionHTTP(t, func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "https://api.github.com/repos/singnet/snet-daemon/releases/latest", r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v1.2.3"}`))}, nil
	})
	for _, tag := range []string{"v1.2.2", "v1.2.3", ""} {
		t.Run("version="+tag, func(t *testing.T) {
			versionTag = tag
			message, err := CheckVersionOfDaemon()
			require.Equal(t, "Daemon version is "+tag, message)
			if tag == "v1.2.2" {
				require.ErrorContains(t, err, "there is a newer version of the Daemon v1.2.3")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetLatestDaemonVersionTransportError(t *testing.T) {
	stubVersionHTTP(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection unavailable")
	})
	version, err := GetLatestDaemonVersion()
	require.Empty(t, version)
	require.ErrorContains(t, err, "error getting latest daemon version from github")
	require.ErrorContains(t, err, "connection unavailable")
}
