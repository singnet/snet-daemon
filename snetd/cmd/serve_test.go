package cmd

import (
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDaemonPort(t *testing.T) {
assert.Equal(t,config.GetString(config.DaemonListeningPort),"8080")
}
