package ratelimit

import (
	"github.com/singnet/snet-daemon/v6/config"
	"golang.org/x/time/rate"
	"math"
	"strconv"
	"time"
)

func NewRateLimiter() *rate.Limiter {
	// Please note that the burst size is ignored when getLimit() returns rate is infinity
	// By Default set the maximum value possible for the Burst Size
	// (assuming rate was defined, but burst was not defined)
	burstSize := config.GetInt(config.BurstSize)
	if burstSize == 0 {
		burstSize = math.MaxInt32
	}
	return rate.NewLimiter(getLimit(), burstSize)
}

func getLimit() rate.Limit {

	ratePerMin, err := strconv.ParseFloat(config.GetString(config.RateLimitPerMinute), 32)
	if err != nil {
		return rate.Inf
	}

	//If the rate limit parameter Value is not defined we will assume it to be Infinity ( no Rate Limiting) ,
	if ratePerMin == 0 {
		return rate.Inf
	}
	intervalMsec := time.Duration((1/float64(ratePerMin))*60*1000) * time.Millisecond
	return rate.Every(intervalMsec)
}
