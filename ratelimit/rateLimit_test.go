package ratelimit

import (
	assert2 "github.com/stretchr/testify/assert"
	"math"
	"testing"
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
