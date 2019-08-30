package cmd

import (

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComponents_verifyMeteringConfigurations(t *testing.T) {
	component := &Components{}
	ok, err := component.verifyMeteringConfigurations("http://demo8325345.mockable.io/verify",
		"testgroup")
	assert.Nil(t,err)
	assert.True(t,ok)

	ok, err = component.verifyMeteringConfigurations("http://demo8325345.mockable.io/badurl","");
	if err != nil {
		assert.Equal(t,err.Error(),"Service call failed with status code : 404 ")
		assert.False(t,ok)

	}

	ok, err = component.verifyMeteringConfigurations("http://demo8325345.mockable.io/failedresponse","")
	if err != nil {
		assert.Equal(t, "Error returned by by Metering Service http://demo8325345.mockable.io/verify Verification, "+
			"pls check the pvt_key_for_metering set up. The public key in metering does not correspond to the private key in Daemon config.", err.Error())
		assert.False(t, ok)
	}

}
