// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"net/http"
	"reflect"
	"testing"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name  string
		state Status
		want  string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getEpochTime(t *testing.T) {
	tests := []struct {
		name string
		want int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEpochTime(); got != tt.want {
				t.Errorf("getEpochTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHeartbeat(t *testing.T) {
	tests := []struct {
		name  string
		want  DaemonHeartbeat
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetHeartbeat()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHeartbeat() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetHeartbeat() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_heartbeatHandler(t *testing.T) {
	type args struct {
		rw http.ResponseWriter
		r  *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heartbeatHandler(tt.args.rw, tt.args.r)
		})
	}
}
