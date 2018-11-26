package cmd

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestDeriveDaemonPort(t *testing.T) {
	port1 := deriveDaemonPort("127.0.0.1:8111")
	assert.Equal(t, "8111", port1)
	port1 = deriveDaemonPort("http://127.0.0.1:8111")
	assert.Equal(t, "8111", port1)
	port1 = deriveDaemonPort("Junk")
	assert.Equal(t, "8080", port1)
}
