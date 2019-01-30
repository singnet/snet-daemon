package cmd

import (
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"testing"
)
//todo
func TestDaemonPort(t *testing.T) {
assert.Equal(t,config.GetString(config.DaemonEndPoint),"127.0.0.1:8080")
}
