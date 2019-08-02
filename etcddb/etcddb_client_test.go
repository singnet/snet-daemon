package etcddb

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func Test_checkIfHttps(t *testing.T) {
	endpoint := []string{"https://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t,checkIfHttps(endpoint),true)
	endpoint = []string{"http://snet-etcd.singularitynet.io:2379"}
	assert.Equal(t,checkIfHttps(endpoint),false)


}
