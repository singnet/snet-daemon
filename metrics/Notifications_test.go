// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import "testing"

func TestNotification_Send(t *testing.T) {
	type fields struct {
		DaemonID  string
		Timestamp string
		Recipient string
		Message   string
		Details   string
		Component string
		Type      string
		Level     string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Notification{
				DaemonID:  tt.fields.DaemonID,
				Timestamp: tt.fields.Timestamp,
				Recipient: tt.fields.Recipient,
				Message:   tt.fields.Message,
				Details:   tt.fields.Details,
				Component: tt.fields.Component,
				Type:      tt.fields.Type,
				Level:     tt.fields.Level,
			}
			if got := alert.Send(); got != tt.want {
				t.Errorf("Notification.Send() = %v, want %v", got, tt.want)
			}
		})
	}
}
