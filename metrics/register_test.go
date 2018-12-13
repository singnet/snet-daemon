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

func Test_callRegisterService(t *testing.T) {
	type args struct {
		daemonID   string
		serviceURL string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotStatus := callRegisterService(tt.args.daemonID, tt.args.serviceURL); gotStatus != tt.wantStatus {
				t.Errorf("callRegisterService() = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}
