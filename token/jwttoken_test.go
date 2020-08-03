package token

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/config"
	"github.com/stretchr/testify/assert"
)

func Test_customJWTokenClaimsImpl_CreateToken(t *testing.T) {
	tokenImpl := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID"
		},
	}
	token, err := tokenImpl.CreateToken(big.NewInt(10))
	assert.Nil(t, err)
	assert.NotNil(t, token)
	tokenInBytes := []byte(fmt.Sprintf("%v", token))
	err = tokenImpl.VerifyToken(string(tokenInBytes), big.NewInt(10))
	config.Vip().Set(config.TokenExpiryInMinutes, 0.1)
	token, err = tokenImpl.CreateToken("any struct")
	time.Sleep(time.Second * 5)
	assert.Nil(t, err)
	err = tokenImpl.VerifyToken(token, "any struct")
	assert.Equal(t, "Token is expired", err.Error())

}

func Test_customJWTokenClaimsImpl_checkJwtTokenClaims(t *testing.T) {
	tokenImpl := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID"
		},
	}
	config.Vip().Set(config.TokenExpiryInMinutes, 0.1)
	token, err := tokenImpl.CreateToken("any struct")
	err = tokenImpl.VerifyToken(token, "different struct")
	assert.Equal(t, "payload any struct used to generate the Token doesnt match expected values", err.Error())
	config.Vip().Set(config.OrganizationId, "differentOrganization")
	err = tokenImpl.VerifyToken(token, "any struct")
	assert.Equal(t, "organization ExampleOrganizationId is not associated with this Daemon", err.Error())
	config.Vip().Set(config.OrganizationId, "ExampleOrganizationId")
	tokenImpl2 := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID2"
		},
	}
	err = tokenImpl2.VerifyToken(token, "any struct")
	assert.Equal(t, "groupId GroupID is not associated with this Daemon", err.Error())
}
