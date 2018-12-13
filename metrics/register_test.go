// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"testing"
)

func Test_registerNewDaemon(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotStatus := registerNewDaemon(); gotStatus != tt.wantStatus {
				t.Errorf("registerNewDaemon() = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func Test_getDaemonID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDaemonID(); got != tt.want {
				t.Errorf("getDaemonID() = %v, want %v", got, tt.want)
			}
		})
	}
}
