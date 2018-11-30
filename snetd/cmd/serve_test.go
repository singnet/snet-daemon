package cmd

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestDeriveDaemonPort(t *testing.T) {
	port1, err := deriveDaemonPort("127.0.0.1:8111")
	assert.Equal(t, nil, err)
	assert.Equal(t, "8111", port1)

	port1, err = deriveDaemonPort("http://127.0.0.1:8111")
	assert.Equal(t, "daemon end point should have a single ':' ,the daemon End point http://127.0.0.1:8111", err.Error())

	port1, err = deriveDaemonPort("abcdefjl:wewee")
	assert.Equal(t, "port number <wewee> is not valid ,the daemon End point  abcdefjl:wewee", err.Error())

	port1, err = deriveDaemonPort("localhost:80:500")
	assert.Equal(t, "daemon end point should have a single ':' ,the daemon End point localhost:80:500", err.Error())

	port1, err = deriveDaemonPort("")
	assert.Equal(t, port1, "8080")
	assert.Equal(t, nil, err)

	port1, err = deriveDaemonPort("127.0.0.1:")
	assert.Equal(t, "port number <> is not valid ,the daemon End point  127.0.0.1:", err.Error())

	port1, err = deriveDaemonPort("127.0.0.1")
	assert.Equal(t, port1, "8080")
	assert.Equal(t, nil, err)
}
