package token

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTTokenServiceCreatesTokensForOrganizationGroup(t *testing.T) {
	originalBlockchainEnabled := config.GetBool(config.BlockchainEnabledKey)
	originalOrganizationID := config.GetString(config.OrganizationId)
	originalSecret := config.GetString(config.TokenSecretKey)
	originalExpiry := config.GetInt(config.TokenExpiryInMinutes)
	config.Vip().Set(config.BlockchainEnabledKey, false)
	config.Vip().Set(config.OrganizationId, "test-org")
	config.Vip().Set(config.TokenSecretKey, "test-secret")
	config.Vip().Set(config.TokenExpiryInMinutes, 1)
	t.Cleanup(func() {
		config.Vip().Set(config.BlockchainEnabledKey, originalBlockchainEnabled)
		config.Vip().Set(config.OrganizationId, originalOrganizationID)
		config.Vip().Set(config.TokenSecretKey, originalSecret)
		config.Vip().Set(config.TokenExpiryInMinutes, originalExpiry)
	})

	organization := blockchain.GetOrganizationMetaData()
	service := NewJWTTokenService(*organization)
	token, err := service.CreateToken("payload", "0xabc")
	require.NoError(t, err)

	address, err := service.VerifyToken(token, "payload")
	require.NoError(t, err)
	require.Equal(t, "0xabc", address)
}

func TestVerifyTokenRejectsNonHMACAlgorithm(t *testing.T) {
	service := customJWTokenServiceImpl{getGroupId: func() string { return "group" }}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{})
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = service.VerifyToken(tokenString, "payload")
	require.ErrorContains(t, err, "unexpected signing method: none")
}

func TestVerifyTokenReturnsUnknownForMissingUserAddress(t *testing.T) {
	originalOrganizationID := config.GetString(config.OrganizationId)
	originalSecret := config.GetString(config.TokenSecretKey)
	config.Vip().Set(config.OrganizationId, "test-org")
	config.Vip().Set(config.TokenSecretKey, "test-secret")
	t.Cleanup(func() {
		config.Vip().Set(config.OrganizationId, originalOrganizationID)
		config.Vip().Set(config.TokenSecretKey, originalSecret)
	})

	service := customJWTokenServiceImpl{getGroupId: func() string { return "group" }}
	token, err := service.CreateToken("payload", "")
	require.NoError(t, err)

	address, err := service.VerifyToken(token, "payload")
	require.NoError(t, err)
	require.Equal(t, "unknown", address)
}

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
	assert.Equal(t, "payload any struct used to generate the token doesn't match expected values", err.Error())
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
	createClaims := func(payload any, orgId string, groupId string) jwt.MapClaims {
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
	assert.EqualError(t, err, "payload payload1 used to generate the token doesn't match expected values")

	// invalid orgId
	claims = createClaims("payload1", "Org2", "GroupID")
	err = tokenImpl.checkJwtTokenClaims(claims, "payload1")
	assert.EqualError(t, err, "organization Org2 is not associated with this Daemon")

	// invalid groupId
	claims = createClaims("payload1", "Org1", "GroupID2")
	err = tokenImpl.checkJwtTokenClaims(claims, "payload1")
	assert.EqualError(t, err, "groupId GroupID2 is not associated with this Daemon")
}
