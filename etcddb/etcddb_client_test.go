package etcddb

import (
	"testing"

	"github.com/magiconair/properties/assert"
	_ "github.com/singnet/snet-daemon/fix-proto"
	"github.com/singnet/snet-daemon/utils"
)

func Test_checkIfHttps(t *testing.T) {
	endpoint := []string{"https://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t, utils.CheckIfHttps(endpoint), true)
	endpoint = []string{"http://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t, utils.CheckIfHttps(endpoint), false)
}
