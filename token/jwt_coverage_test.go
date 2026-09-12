package token

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
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
