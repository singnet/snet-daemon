package etcddb

import (
	"github.com/magiconair/properties/assert"
	"github.com/singnet/snet-daemon/utils"
	"testing"
)

func Test_checkIfHttps(t *testing.T) {
	endpoint := []string{"https://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t, utils.CheckIfHttps(endpoint), true)
	endpoint = []string{"http://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t, utils.CheckIfHttps(endpoint), false)

}
