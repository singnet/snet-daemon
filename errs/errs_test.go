package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrDescURL(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{ServiceUnavailable, "\nAbout error & possible fixes: " + devPortalURL + "1"},
		{InvalidMetadata, "\nAbout error & possible fixes: " + devPortalURL + "2"},
		{InvalidProto, "\nAbout error & possible fixes: " + devPortalURL + "3"},
		{HTTPRequestBuildError, "\nAbout error & possible fixes: " + devPortalURL + "4"},
		{InvalidServiceCredentials, "\nAbout error & possible fixes: " + devPortalURL + "5"},
		{InvalidConfig, "\nAbout error & possible fixes: " + devPortalURL + "6"},
		{ReceiveMsgError, "\nAbout error & possible fixes: " + devPortalURL + "7"},
		{BlockchainProviderLimitsExceed, "\nAbout error & possible fixes: " + devPortalURL + "8"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, ErrDescURL(tt.code))
		})
	}
}

func TestErrorCodeValues(t *testing.T) {
	assert.Equal(t, 1, ServiceUnavailable)
	assert.Equal(t, 2, InvalidMetadata)
	assert.Equal(t, 3, InvalidProto)
	assert.Equal(t, 4, HTTPRequestBuildError)
	assert.Equal(t, 5, InvalidServiceCredentials)
	assert.Equal(t, 6, InvalidConfig)
	assert.Equal(t, 7, ReceiveMsgError)
	assert.Equal(t, 8, BlockchainProviderLimitsExceed)
}
