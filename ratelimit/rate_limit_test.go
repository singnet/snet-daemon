package ratelimit

import (
	"math"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	assert2 "github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestGetLimitInvalidConfig(t *testing.T) {
	original := config.GetString(config.RateLimitPerMinute)
	config.Vip().Set(config.RateLimitPerMinute, "not_a_number")
	defer config.Vip().Set(config.RateLimitPerMinute, original)

	assert2.Equal(t, rate.Inf, getLimit())
}

func TestGetLimitZeroReturnsInfinity(t *testing.T) {
	original := config.GetString(config.RateLimitPerMinute)
	config.Vip().Set(config.RateLimitPerMinute, "0")
	defer config.Vip().Set(config.RateLimitPerMinute, original)

	assert2.Equal(t, rate.Inf, getLimit())
}

func TestGetLimitWithRate(t *testing.T) {
	original := config.GetString(config.RateLimitPerMinute)
	config.Vip().Set(config.RateLimitPerMinute, "60")
	defer config.Vip().Set(config.RateLimitPerMinute, original)

	limit := getLimit()
	expected := rate.Every(time.Duration(1e9))
	assert2.Equal(t, expected, limit)
}

func TestNewRateLimiterDefaultBurst(t *testing.T) {
	rateLimit := NewRateLimiter()
	assert2.Equal(t, math.MaxInt32, rateLimit.Burst())
}

func TestNewRateLimiterCustomBurst(t *testing.T) {
	originalBurst := config.GetInt(config.BurstSize)
	config.Vip().Set(config.BurstSize, 10)
	defer config.Vip().Set(config.BurstSize, originalBurst)

	rateLimit := NewRateLimiter()
	assert2.Equal(t, 10, rateLimit.Burst())
}
