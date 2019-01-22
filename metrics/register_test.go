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
	assert.Equal(t, "10990b62daf504a0ae6094d548f25aed4928b1e991a9221a31693890c20d6916", daemonID)
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
