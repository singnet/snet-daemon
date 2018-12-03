package ratelimit

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestGetRateLimiter(t *testing.T) {
	limit := getLimit()
	assert.Equal(t, int(limit), 1)

}
