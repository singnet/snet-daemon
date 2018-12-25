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
	err := fmt.Errorf("TEst Error")
	msg := getErrorMessage(err)
	assert.Equal(t, msg, "TEst Error")
	assert.Equal(t, getErrorMessage(nil), "")

}

func TestGetErrorCode(t *testing.T) {
	err := fmt.Errorf("TEst Error")
	code := getErrorCode(err)
	assert.Equal(t, code, "Unknown")
	assert.Equal(t, getErrorCode(nil), "OK")
}
