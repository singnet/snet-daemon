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

func TestCreateNotification(t *testing.T) {
	arrivalTime := time.Now()
	err := fmt.Errorf("test Error")
	commonStat := BuildCommonStats(arrivalTime, "TestMethod")
	response := createResponseStats(commonStat, time.Duration(12), err)
	notification := createNotification(response)
	assert.Equal(t, notification.DaemonID, GetDaemonID())
	assert.Equal(t, notification.Details, "test Error")
	assert.Equal(t, notification.Message, "Error on call of Service Method :TestMethod")
	assert.Equal(t, notification.Timestamp, response.ResponseSentTime)
}
