package cmd

import (
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

func TestComponents_verifyMeteringConfigurations(t *testing.T) {
	config.Vip().Set(config.MeteringEndpoint, "http://demo5343751.mockable.io")

	component := &Components{blockchain: blockchain.NewMockProcessor(true)}
	ok, err := component.verifyAuthenticationSetUpForFreeCall("http://demo5343751.mockable.io/verify",
		"testgroup")
	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo5343751.mockable.io/test", "")
	if err != nil {
		assert.Equal(t, err.Error(), "you need a specify a valid private key 'pvt_key_for_metering' as part of service publication process.invalid length, need 256 bits")
		assert.False(t, ok)

	}
	config.Vip().Set(config.PvtKeyForMetering, "6996606c7854992c10d8cdc9a13d511a9d9db8ab8f21e59d6ac901a76367b36b")
	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo5343751.mockable.io/verify",
		"testgroup")
	assert.NotNil(t, err)
	assert.False(t, ok)
	//todo , bring a local service to validate the auth.

	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo5343751.mockable.io/badurl", "")
	if err != nil {
		assert.Equal(t, err.Error(), "Service call failed with status code : 404 ")
		assert.False(t, ok)

	}

	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo5343751.mockable.io/failedresponse", "")
	if err != nil {
		//todo , bring up a local end point to test this
		/*assert.Equal(t, "Error returned by by Metering Service http://demo5343751.mockable.io/verify Verification, "+
		"pls check the pvt_key_for_metering set up. The public key in metering does not correspond to the private key in Daemon config.", err.Error())*/
		assert.False(t, ok)
	}

}
