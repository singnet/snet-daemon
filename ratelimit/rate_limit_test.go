package ratelimit

import (
	"math"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

// TO DO , Add more test cases
func TestGetRateLimiter(t *testing.T) {
	limit := getLimit()
	assert2.NotEqual(t, nil, limit)
}

func TestNewRateLimiter(t *testing.T) {
	rateLimit := NewRateLimiter()
	assert2.Equal(t, rateLimit.Burst(), math.MaxInt32)
}
