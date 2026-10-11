package config

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestGetLatestDaemonVersionErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{name: "unexpected status", statusCode: http.StatusTooManyRequests, body: `{}`, wantErr: "github releases API returned status 429 Too Many Requests"},
		{name: "invalid JSON", statusCode: http.StatusOK, body: `{`, wantErr: "error decoding latest daemon version"},
		{name: "missing tag", statusCode: http.StatusOK, body: `{}`, wantErr: "github releases API response has no tag_name"},
		{name: "empty tag", statusCode: http.StatusOK, body: `{"tag_name":""}`, wantErr: "github releases API response has no tag_name"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			version, err := getLatestDaemonVersion(server.Client(), server.URL)
			require.Empty(t, version)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestGetLatestDaemonVersionSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/releases/latest" {
			http.Error(w, "unexpected release request", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.3"}`)
	}))
	t.Cleanup(server.Close)

	version, err := getLatestDaemonVersion(server.Client(), server.URL+"/releases/latest")
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", version)
}

func TestGetLatestDaemonVersionTimeout(t *testing.T) {
	require.Equal(t, 10*time.Second, versionHTTPClient.Timeout)
	for _, tc := range []struct {
		name      string
		flushBody bool
		wantErr   string
	}{
		{name: "waiting for headers", wantErr: "error getting latest daemon version from github"},
		{name: "reading body", flushBody: true, wantErr: "error reading latest daemon version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.flushBody {
					_, _ = io.WriteString(w, `{"tag_name":`)
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			t.Cleanup(server.Close)
			client := server.Client()
			client.Timeout = 250 * time.Millisecond

			version, err := getLatestDaemonVersion(client, server.URL)
			require.Empty(t, version)
			require.ErrorContains(t, err, tc.wantErr)
			require.ErrorIs(t, err, context.DeadlineExceeded)
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
	wantErr := errors.New("connection unavailable")
	stubVersionHTTP(t, func(*http.Request) (*http.Response, error) {
		return nil, wantErr
	})
	version, err := GetLatestDaemonVersion()
	require.Empty(t, version)
	require.ErrorContains(t, err, "error getting latest daemon version from github")
	require.ErrorContains(t, err, "connection unavailable")
	require.ErrorIs(t, err, wantErr)
}
