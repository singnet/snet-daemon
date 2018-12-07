package ratelimit

import (
	"github.com/singnet/snet-daemon/config"
	"golang.org/x/time/rate"
	"math"
	"time"
)

func NewRateLimiter() rate.Limiter {
	//Please note that the burst size is ignored when getLimit() returns rate is infinity
	//By Default set the maximum value possible for the Burst Size ( assuming rate was defined ,but burst was not defined)
	burstSize := config.GetInt(config.BurstSize)
	if burstSize == 0 {
		burstSize = math.MaxInt64
	}
	limiter := rate.NewLimiter(getLimit(), burstSize)
	return *limiter
}

func getLimit() rate.Limit {
	ratePerMin := config.GetInt(config.RateLimitPerMinute)
	//If the rate limit parameter Value is not defined we will assume it to be Infinity ( no Rate Limiting) ,
	if ratePerMin == 0 {
		return rate.Inf
	}
	intervalMsec := time.Duration((1/float64(ratePerMin))*60*1000) * time.Millisecond
	return rate.Every(intervalMsec)
}
