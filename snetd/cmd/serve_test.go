package cmd

import (
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

// TODO
func TestDaemonPort(t *testing.T) {
	assert.Equal(t, config.GetString(config.DaemonEndpoint), "127.0.0.1:8080")
}
