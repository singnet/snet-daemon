package utils

import "testing"

func TestCheckIfHttpsUsesURLScheme(t *testing.T) {
	for _, tc := range []struct {
		endpoint string
		want     bool
	}{
		{"https://example.com:2379", true},
		{"http://https.example.com:2379", false},
		{"http://example.com/https", false},
		{"http://example.com?transport=https", false},
		{"https-not-a-url", false},
		{"https://", false},
	} {
		if got := CheckIfHttps([]string{tc.endpoint}); got != tc.want {
			t.Errorf("CheckIfHttps(%q) = %v, want %v", tc.endpoint, got, tc.want)
		}
	}
}
