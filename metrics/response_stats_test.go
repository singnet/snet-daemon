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
	response := createResponseStats(commonStat, time.Duration(12), nil)
	assert.Equal(t, response.RequestID, commonStat.ID)
	assert.Equal(t, response.GroupID, daemonGroupId)
	assert2.NotEqual(t, response.ResponseSentTime, "")
}

func TestGetErrorMessage(t *testing.T) {
	err := fmt.Errorf("Test Error")
	msg := getErrorMessage(err)
	assert.Equal(t, msg, "Test Error")
	assert.Equal(t, getErrorMessage(nil), "")

}

func TestGetErrorCode(t *testing.T) {
	err := fmt.Errorf("Test Error")
	code := getErrorCode(err)
	assert.Equal(t, code, "Unknown")
	assert.Equal(t, getErrorCode(nil), "OK")
}

func TestCreateNotificaiton(t *testing.T) {
	arrivalTime := time.Now()
	err := fmt.Errorf("Test Error")
	commonStat := BuildCommonStats(arrivalTime, "TestMethod")
	response := createResponseStats(commonStat, time.Duration(12), err)
	notification := createNotification(response)
	assert.Equal(t, notification.DaemonID, GetDaemonID())
	assert.Equal(t, notification.Details, "Test Error")
	assert.Equal(t, notification.Message, "Error on call of Service Method :TestMethod")
	assert.Equal(t, notification.Timestamp, response.ResponseSentTime)
}
