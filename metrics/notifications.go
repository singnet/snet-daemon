// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"errors"
	"math/big"

	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

// state of alerts configuration
var isNoAlertsConfig bool

// define heartbeat data model. Service Status JSON object Array marshalled to a string
type Notification struct {
	DaemonID  string `json:"component_id"`
	Timestamp string `json:"timestamp"`
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	Details   string `json:"details"`
	Component string `json:"component"`
	Type      string `json:"type"`
	Level     string `json:"level"`
}

// function for sending an alert to a given endpoint
func (alert *Notification) Send(currentBlock *big.Int) bool {
	if isNoAlertsConfig {
		zap.L().Warn("notifications not configured")
		return false
	}
	serviceURL := config.GetString(config.NotificationServiceEndpoint)
	// convert the notification struct to json
	jsonAlert, err := ConvertStructToJSON(alert)
	zap.L().Info("send notification", zap.Any("json", jsonAlert))
	if err != nil {
		zap.L().Warn("json conversion error", zap.Error(err))
		return false
	} else {
		// based on the notification success/failure
		status := Publish(jsonAlert, serviceURL, &CommonStats{}, currentBlock)
		if !status {
			zap.L().Info("unable to send notifications")
			return false
		}
	}
	return true
}

// set the no alerts URL and email State
func SetIsNoAlertsConfig(state bool) {
	isNoAlertsConfig = state
}

// validates the heartbeat configurations
func ValidateNotificationConfig() error {
	SetIsNoAlertsConfig(false)

	// if the URL or email is empty, consider it as not configured and set isInvalidAlertsConfig to true
	if config.GetString(config.NotificationServiceEndpoint) == "" || config.GetString(config.AlertsEMail) == "" {
		SetIsNoAlertsConfig(true)
	} else if !(config.IsValidUrl(config.GetString(config.NotificationServiceEndpoint)) &&
		config.ValidateEmail(config.GetString(config.AlertsEMail))) {
		return errors.New("service endpoint  and alerts email id must be valid")
	}
	return nil
}

/*
service request
{
    "recipient":"raam.comm@gmail.com",
    "message":"From the API",
    "details":"From the API",
    "component":"daemon",
    "component_id":"ad",
    "type":"INFO",
    "level":"10"
}

service response
true/false
*/
