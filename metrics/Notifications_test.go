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
		To        string
		Message   string
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
				To:        tt.fields.To,
				Message:   tt.fields.Message,
			}
			if got := alert.Send(); got != tt.want {
				t.Errorf("Notification.Send() = %v, want %v", got, tt.want)
			}
		})
	}
}
