package cmd

import (
	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComponents_verifyMeteringConfigurations(t *testing.T) {
	config.Vip().Set(config.MeteringEndPoint,"http://demo8325345.mockable.io")
	component := &Components{}
	ok, err := component.verifyAuthenticationSetUpForFreeCall("http://demo8325345.mockable.io/verify",
		"testgroup")
	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo8325345.mockable.io/badurl","");
	if err != nil {
		assert.Equal(t,err.Error(),"you need a specify a valid private key 'pvt_key_for_metering' " +
			"given by you as part of curation process to support free calls invalid length, need 256 bits")
		assert.False(t,ok)

	}
	config.Vip().Set(config.PvtKeyForMetering,"6996606c7854992c10d8cdc9a13d511a9d9db8ab8f21e59d6ac901a76367b36b")
	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo8325345.mockable.io/verify",
		"testgroup")
	assert.Nil(t,err)
	assert.True(t,ok)

	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo8325345.mockable.io/badurl","");
	if err != nil {
		assert.Equal(t,err.Error(),"Service call failed with status code : 404 ")
		assert.False(t,ok)

	}

	ok, err = component.verifyAuthenticationSetUpForFreeCall("http://demo8325345.mockable.io/failedresponse","")
	if err != nil {
		assert.Equal(t, "Error returned by by Metering Service http://demo8325345.mockable.io/verify Verification, "+
			"pls check the pvt_key_for_metering set up. The public key in metering does not correspond to the private key in Daemon config.", err.Error())
		assert.False(t, ok)
	}

}
