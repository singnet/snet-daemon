package ratelimit

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
)

//TO DO , Add more test cases
func TestGetRateLimiter(t *testing.T) {
	limit := getLimit()
	assert2.NotEqual(t, nil, limit)
}
