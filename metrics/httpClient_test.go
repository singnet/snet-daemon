// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"testing"
)

func Test_isValidUrl(t *testing.T) {
	type args struct {
		urlToTest string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUrl(tt.args.urlToTest); got != tt.want {
				t.Errorf("isValidUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getServiceHeartbeat(t *testing.T) {
	type args struct {
		serviceURL string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getServiceHeartbeat(tt.args.serviceURL)
			if got != tt.want {
				t.Errorf("getServiceHeartbeat() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getServiceHeartbeat() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
