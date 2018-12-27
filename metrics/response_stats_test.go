package metrics

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	assert2 "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCreateResponseStats(t *testing.T) {
	arrivalTime := time.Now()
	commonStat := BuildCommonStats(arrivalTime, "TestMethod")
	response := createResponseStats(commonStat, time.Duration(1234566000), nil)
	assert.Equal(t, response.RequestID, commonStat.ID)
	assert.Equal(t, response.GroupID, daemonGroupId)
	assert.Equal(t, response.ResponseTime, "1.2346")
	assert2.NotEqual(t, response.ResponseSentTime, "")
}

func TestGetErrorMessage(t *testing.T) {
	err := fmt.Errorf("test Error")
	msg := getErrorMessage(err)
	assert.Equal(t, msg, "test Error")
	assert.Equal(t, getErrorMessage(nil), "")

}

func TestGetErrorCode(t *testing.T) {
	err := fmt.Errorf("test Error")
	code := getErrorCode(err)
	assert.Equal(t, code, "Unknown")
	assert.Equal(t, getErrorCode(nil), "OK")
}
