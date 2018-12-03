package ratelimit

import (
	"github.com/singnet/snet-daemon/config"
	"golang.org/x/time/rate"
	"time"
)

func GetRateLimiter() rate.Limiter {
	limiter := rate.NewLimiter(getLimit(), config.GetInt(config.BurstSize)) //separate config for Burst size ? or derive this from ratelimit ( say twice the rate limit) ?
	return *limiter
}

func getLimit() rate.Limit {
	configRateLimit := config.GetInt(config.RateLimitPerMinute)
	ratePerMin := float64(configRateLimit)
	intervalMsec := time.Duration((1/ratePerMin)*60*1000) * time.Millisecond
	return rate.Every(intervalMsec)
}
