package token

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
)

func Test_customJWTokenClaimsImpl_CreateToken(t *testing.T) {
	tokenImpl := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID"
		},
	}
	token, err := tokenImpl.CreateToken(big.NewInt(10), "0x")
	assert.Nil(t, err)
	assert.NotNil(t, token)
	address, err := tokenImpl.VerifyToken(fmt.Sprintf("%v", token), big.NewInt(10))
	assert.Equal(t, "0x", address)
	config.Vip().Set(config.TokenExpiryInMinutes, 0.1)
	token, err = tokenImpl.CreateToken("any struct", "0x")
	time.Sleep(time.Second * 5)
	assert.Nil(t, err)
	_, err = tokenImpl.VerifyToken(token, "any struct")
	assert.Equal(t, "token has invalid claims: token is expired", err.Error())
}

func Test_customJWTokenClaimsImpl_checkJwtTokenClaims(t *testing.T) {
	tokenImpl := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID"
		},
	}
	config.Vip().Set(config.TokenExpiryInMinutes, 1)
	token, err := tokenImpl.CreateToken("any struct", "0x")
	_, err = tokenImpl.VerifyToken(token, "different struct")
	assert.Equal(t, "payload any struct used to generate the Token doesnt match expected values", err.Error())
	config.Vip().Set(config.OrganizationId, "differentOrganization")
	_, err = tokenImpl.VerifyToken(token, "any struct")
	assert.Equal(t, "organization YOUR_ORG_ID is not associated with this Daemon", err.Error())
	config.Vip().Set(config.OrganizationId, "YOUR_ORG_ID")
	tokenImpl2 := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID2"
		},
	}
	_, err = tokenImpl2.VerifyToken(token, "any struct")
	assert.Equal(t, "groupId GroupID is not associated with this Daemon", err.Error())
}

func Test_customJWTokenServiceImpl_checkJwtTokenClaims(t *testing.T) {
	tokenImpl := &customJWTokenServiceImpl{
		getGroupId: func() string {
			return "GroupID"
		},
	}

	// helper to create the claims map
	createClaims := func(payload interface{}, orgId string, groupId string) jwt.MapClaims {
		return jwt.MapClaims{
			"payload": payload,
			"orgId":   orgId,
			"groupId": groupId,
		}
	}

	// valid claims
	claims := createClaims("payload1", "Org1", "GroupID")
	tokenImpl.getGroupId = func() string { return "GroupID" }
	config.Vip().Set(config.OrganizationId, "Org1")
	err := tokenImpl.checkJwtTokenClaims(claims, "payload1")
	assert.NoError(t, err)

	// invalid payload
	claims = createClaims("payload1", "Org1", "GroupID")
	err = tokenImpl.checkJwtTokenClaims(claims, "payload2")
	assert.EqualError(t, err, "payload payload1 used to generate the Token doesnt match expected values")

	// invalid orgId
	claims = createClaims("payload1", "Org2", "GroupID")
	err = tokenImpl.checkJwtTokenClaims(claims, "payload1")
	assert.EqualError(t, err, "organization Org2 is not associated with this Daemon")

	// invalid groupId
	claims = createClaims("payload1", "Org1", "GroupID2")
	err = tokenImpl.checkJwtTokenClaims(claims, "payload1")
	assert.EqualError(t, err, "groupId GroupID2 is not associated with this Daemon")
}
