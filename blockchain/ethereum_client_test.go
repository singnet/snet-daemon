package blockchain

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBasicAuth checks basicAuth encoding
func TestBasicAuth(t *testing.T) {
	encoded := basicAuth("user", "pass")
	expected := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	assert.Equal(t, expected, encoded)

	encodedEmptyUser := basicAuth("", "key")
	expectedEmptyUser := base64.StdEncoding.EncodeToString([]byte(":key"))
	assert.Equal(t, expectedEmptyUser, encodedEmptyUser)
}

func TestGetAuthOption(t *testing.T) {
	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature"
	classicKey := "classicSecretKey"

	tests := []struct {
		name     string
		endpoint string
		apiKey   string
		wantNil  bool
	}{
		{"empty apiKey", "https://infura.io", "", true},
		{"jwt token", "https://infura.io", jwtToken, false},
		{"infura classic key", "https://infura.io", classicKey, false},
		{"alchemy key", "https://alchemy.com", classicKey, false},
		{"other provider", "https://example.com", classicKey, false},
		{"other provider", "https://custom.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := getAuthOption(tt.endpoint, tt.apiKey)
			if tt.wantNil && opt != nil {
				t.Errorf("expected nil but got non-nil")
			}
			if !tt.wantNil && opt == nil {
				t.Errorf("expected non-nil but got nil")
			}
		})
	}
}
