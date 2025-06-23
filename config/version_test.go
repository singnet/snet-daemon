package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
	versionTag = "not-latest"
	message, err := CheckVersionOfDaemon()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "there is a newer version of the Daemon")

	versionTag, _ = GetLatestDaemonVersion()
	message, err = CheckVersionOfDaemon()
	assert.Nil(t, err)
	assert.Contains(t, message, "Daemon version is "+versionTag)
}
