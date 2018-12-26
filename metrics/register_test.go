package metrics

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestSetDaemonGrpId(t *testing.T) {
	SetDaemonGrpId("123#$$")
	assert.Equal(t, daemonGroupId, "123#$$")
}
