// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"errors"
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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
	result := notification.Send()
	assert.False(t, result)
}
