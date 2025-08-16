// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

func TestNotification_Send(t *testing.T) {
	err := errors.New("dummy error for mail notification test")
	// send the alert if service heartbeat fails
	notification := &Notification{
		Recipient: config.GetString(config.AlertsEMail),
		Details:   err.Error(),
		Timestamp: time.Now().String(),
		Message:   "some random error message",
		Component: "Daemon",
		DaemonID:  GetDaemonID(),
		Level:     "ERROR",
	}
	result := notification.Send(big.NewInt(100))
	assert.False(t, result)
}

func TestSetIsNoAlertsConfig(t *testing.T) {
	SetIsNoAlertsConfig(true)
	assert.Equal(t, true, isNoAlertsConfig)

	SetIsNoAlertsConfig(false)
	assert.Equal(t, false, isNoAlertsConfig)
}

func TestValidateNotificationConfig(t *testing.T) {
	err := ValidateNotificationConfig()
	assert.Nil(t, err)
	assert.Equal(t, true, isNoAlertsConfig) //because default email is empty
}
