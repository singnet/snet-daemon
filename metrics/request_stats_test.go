package metrics

import (
	"github.com/magiconair/properties/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestSetDataFromContext(t *testing.T) {
	md := metadata.Pairs("user-agent", "Test user agent", "time", "2018-09-93", "content-type", "application/grpc")
	request := &RequestStats{}
	setDataFromContext(md, request)
	assert.Equal(t, request.UserAgent, "Test user agent")
	assert.Equal(t, request.RequestArrivalTime, "2018-09-93")
	assert.Equal(t, request.ContentType, "application/grpc")

}

func TestCreateRequestStat(t *testing.T) {
	request := createRequestStat("123", "A1234")
	assert.Equal(t, request.RequestID, "123")
	assert.Equal(t, request.GroupID, "A1234")
}
