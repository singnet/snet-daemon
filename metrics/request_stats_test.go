package metrics

import (
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

//func TestSetDataFromContext(t *testing.T) {
//	md := metadata.Pairs("user-agent", "Test user agent", "time", "2018-09-93", "content-type", "application/")
//	request := &RequestStats{}
//	request.setDataFromContext(md)
//}

func TestCreateRequestStat(t *testing.T) {
	arrivalTime := time.Now()
	commonStat := BuildCommonStats(arrivalTime, "TestMethod")
	commonStat.ClientType = "snet-cli"
	commonStat.UserDetails = "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"
	commonStat.UserAgent = "python/cli"
	commonStat.ChannelId = "2"
	request := createRequestStat(commonStat)
	assert.Equal(t, request.RequestID, commonStat.ID)
	assert.Equal(t, request.GroupID, daemonGroupId)
	assert.Equal(t, request.OrganizationID, config.GetString(config.OrganizationId))
	assert.Equal(t, request.ServiceID, config.GetString(config.ServiceId))
	assert.Equal(t, request.RequestReceivedTime, arrivalTime.UTC().Format(timeFormat))
	assert.Equal(t, request.Version, commonStat.Version)
	assert.Equal(t, request.Type, "request")
	assert.Equal(t, request.ClientType, "snet-cli")
	assert.Equal(t, request.UserDetails, "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
	assert.Equal(t, request.UserAgent, "python/cli")
	assert.Equal(t, request.ChannelId, "2")
}
