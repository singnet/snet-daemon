// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDaemonID(t *testing.T) {
	daemonID := GetDaemonID()

	assert.NotNil(t, daemonID, "daemon ID must not be nil")
	assert.NotEmpty(t, daemonID, "daemon ID must not be empty")
	assert.Equal(t, "2188ffe79222a44083c315dbb6bc82f3292fa76131b226a85c8ed11361a2406f", daemonID)
	assert.NotEqual(t, "48d343313a1e06093c81830103b45496cc7c277f321049e9ee632fd03207d042", daemonID)
}

func TestRegisterDaemon(t *testing.T) {
	serviceURL := "https://demo3208027.mockable.io/register"

	result := RegisterDaemon(serviceURL)
	assert.Equal(t, true, result)

	serviceURL = "https://demo3208027.mockable.io/registererror"
	result = RegisterDaemon(serviceURL)
	assert.Equal(t, false, result)
}

func TestSetDaemonGrpId(t *testing.T) {
	grpid := "group01"
	SetDaemonGrpId(grpid)
	assert.NotNil(t, daemonGroupId)
	assert.NotEmpty(t, daemonGroupId)
	assert.Equal(t, "group01", daemonGroupId)
	assert.NotEqual(t, "some wrong group id", daemonGroupId)
}
