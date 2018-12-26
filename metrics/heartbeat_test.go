// Copyright 2018 singularitynet foundation.
// All rights reserved.
// <<add licence terms for code reuse>>

// package for monitoring and reporting the daemon metrics
package metrics

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_String(t *testing.T) {
	assert.Equal(t, Online.String(), "Online", "Invalid enum string conversion")
	assert.NotEqual(t, Online.String(), "Offline", "Invalid enum string conversion")
}

func TestHeartbeatHandler(t *testing.T) {
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
			HeartbeatHandler(tt.args.rw, tt.args.r)
		})
	}
}

func TestGetHeartbeat(t *testing.T) {
	type args struct {
		serviceURL string
	}
	tests := []struct {
		name  string
		args  args
		want  DaemonHeartbeat
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHeartbeat(tt.args.serviceURL)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHeartbeat() got = %v, want %v", got, tt.want)
			}
		})
	}
}
