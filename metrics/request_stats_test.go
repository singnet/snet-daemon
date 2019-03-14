package metrics

import (
	"github.com/magiconair/properties/assert"
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

func TestSetDataFromContext(t *testing.T) {
	md := metadata.Pairs("user-agent", "Test user agent", "time", "2018-09-93", "content-type", "application/")
	request := &RequestStats{}
	request.setDataFromContext(md)
}

func TestCreateRequestStat(t *testing.T) {
	arrivalTime := time.Now()
	commonStat := BuildCommonStats(arrivalTime, "TestMethod")
	request := createRequestStat(commonStat)
	assert.Equal(t, request.RequestID, commonStat.ID)
	assert.Equal(t, request.GroupID, daemonGroupId)
	assert.Equal(t, request.OrganizationID, config.GetString(config.OrganizationId))
	assert.Equal(t, request.ServiceID, config.GetString(config.ServiceId))
	assert.Equal(t, request.RequestReceivedTime, arrivalTime.String())
	assert.Equal(t, request.Type, "request")
}
