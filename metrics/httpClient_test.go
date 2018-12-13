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

func Test_callServiceHeartbeat(t *testing.T) {
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
			got, got1 := callServiceHeartbeat(tt.args.serviceURL)
			if got != tt.want {
				t.Errorf("callServiceHeartbeat() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("callServiceHeartbeat() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_callAndPostMetrics(t *testing.T) {
	type args struct {
		sericeURL   string
		jsonMetrcis string
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
			if got := callAndPostMetrics(tt.args.sericeURL, tt.args.jsonMetrcis); got != tt.want {
				t.Errorf("callAndPostMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}
