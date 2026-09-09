package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

func TestComponentsVerifyMeteringConfigurations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/verify":
			assert.Equal(t, "verification", request.Header.Get("x-authtype"))
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"data":"success"}`))
		case "/failedresponse":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"data":"failed"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	originalEndpoint := config.GetString(config.MeteringEndpoint)
	originalKey := config.GetString(config.PvtKeyForMetering)
	config.Vip().Set(config.MeteringEndpoint, server.URL)
	t.Cleanup(func() {
		config.Vip().Set(config.MeteringEndpoint, originalEndpoint)
		config.Vip().Set(config.PvtKeyForMetering, originalKey)
	})

	component := &Components{blockchain: blockchain.NewMockProcessor(true)}
	ok, err := component.verifyAuthenticationSetUpForFreeCall(server.URL+"/verify", "testgroup")
	assert.EqualError(t, err, "you need a specify a valid private key 'pvt_key_for_metering' as part of service publication process.invalid length, need 256 bits")
	assert.False(t, ok)

	config.Vip().Set(config.PvtKeyForMetering, "6996606c7854992c10d8cdc9a13d511a9d9db8ab8f21e59d6ac901a76367b36b")
	ok, err = component.verifyAuthenticationSetUpForFreeCall(server.URL+"/verify", "testgroup")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = component.verifyAuthenticationSetUpForFreeCall(server.URL+"/badurl", "")
	assert.EqualError(t, err, "Service call failed with status code : 404 ")
	assert.False(t, ok)

	ok, err = component.verifyAuthenticationSetUpForFreeCall(server.URL+"/failedresponse", "")
	assert.EqualError(t, err, "error returned by by Metering Service "+server.URL+"/verify Verification, pls check the pvt_key_for_metering set up. The public key in metering does not correspond to the private key in Daemon config")
	assert.False(t, ok)
}
