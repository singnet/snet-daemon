package metrics

import (
	"github.com/magiconair/properties/assert"
	"github.com/singnet/snet-daemon/config"
	"google.golang.org/grpc/metadata"
	"testing"
	time2 "time"
)

func TestSetDataFromContext(t *testing.T) {
	md := metadata.Pairs("user-agent", "Test user agent", "time", "2018-09-93", "content-type", "application/grpc")
	request := &RequestStats{}
	setDataFromContext(md, request)
	assert.Equal(t, request.UserAgent, "Test user agent")
	assert.Equal(t, request.ContentType, "application/grpc")

}

func TestCreateRequestStat(t *testing.T) {
	time := time2.Now()
	request := createRequestStat("123", "A1234", time)
	assert.Equal(t, request.RequestID, "123")
	assert.Equal(t, request.GroupID, "A1234")
	assert.Equal(t, request.DaemonEndPoint, config.GetString(config.DaemonEndPoint))
	assert.Equal(t, request.OrganizationID, config.GetString(config.OrganizationId))
	assert.Equal(t, request.ServiceID, config.GetString(config.ServiceId))
	assert.Equal(t, request.RequestReceivedTime, time.String())
}
