package metrics

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"testing"
	"time"
)

func TestBuildResponseStats(t *testing.T) {

	response := PublishResponseStats("123", "#we3", time.Duration(12), nil)
	assert.Equal(t, response.RequestID, "123")
	assert.Equal(t, response.GroupID, "#we3")

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
