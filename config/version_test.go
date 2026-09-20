package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
	}{
		{name: "unexpected status", statusCode: http.StatusTooManyRequests, body: `{}`},
		{name: "invalid JSON", statusCode: http.StatusOK, body: `{`},
		{name: "missing tag", statusCode: http.StatusOK, body: `{}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			_, err := getLatestDaemonVersion(server.Client(), server.URL)
			assert.Error(t, err)
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
	versionTag = "not-latest"
	message, err := CheckVersionOfDaemon()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "there is a newer version of the Daemon")

	versionTag, _ = GetLatestDaemonVersion()
	message, err = CheckVersionOfDaemon()
	assert.Nil(t, err)
	assert.Contains(t, message, "Daemon version is "+versionTag)
}
