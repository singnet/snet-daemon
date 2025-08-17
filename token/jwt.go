package token

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
)

type customJWTokenServiceImpl struct {
	getGroupId func() string
}

// NewJWTTokenService service to Create and Validate JWT tokens
func NewJWTTokenService(data blockchain.OrganizationMetaData) Manager {
	return &customJWTokenServiceImpl{
		getGroupId: func() string {
			return data.GetGroupIdString()
		},
	}
}

func (service customJWTokenServiceImpl) CreateToken(payLoad PayLoad, userAddress string) (CustomToken, error) {
	atClaims := jwt.MapClaims{}
	atClaims["payload"] = fmt.Sprintf("%v", payLoad)
	atClaims["userAddress"] = userAddress
	atClaims["orgId"] = config.GetString(config.OrganizationId)
	atClaims["groupId"] = service.getGroupId()
	//set the Expiry of the Token generated
	atClaims["exp"] = time.Now().UTC().
		Add(time.Minute * time.Duration(config.GetInt(config.TokenExpiryInMinutes))).Unix()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	return jwtToken.SignedString([]byte(config.GetString(config.TokenSecretKey)))
}

func (service customJWTokenServiceImpl) VerifyToken(receivedToken CustomToken, payLoad PayLoad) (userAddress string, err error) {
	tokenString := fmt.Sprintf("%v", receivedToken)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.GetString(config.TokenSecretKey)), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	if err := service.checkJwtTokenClaims(claims, payLoad); err != nil {
		return "", err
	}

	senderVal, ok := claims["userAddress"].(string)
	if !ok || senderVal == "" {
		return "unknown", nil
	}

	return senderVal, err
}

func (service customJWTokenServiceImpl) checkJwtTokenClaims(claims jwt.MapClaims, payload PayLoad) (err error) {
	if strings.Compare(fmt.Sprintf("%v", claims["payload"]), fmt.Sprintf("%v", payload)) != 0 {
		return fmt.Errorf("payload %v used to generate the Token doesnt match expected values", claims["payload"])
	}

	if strings.Compare(fmt.Sprintf("%v", claims["orgId"]), config.GetString(config.OrganizationId)) != 0 {
		return fmt.Errorf("organization %v is not associated with this Daemon", claims["orgId"])
	}

	if strings.Compare(fmt.Sprintf("%v", claims["groupId"]), service.getGroupId()) != 0 {
		return fmt.Errorf("groupId %v is not associated with this Daemon", claims["groupId"])
	}
	return nil
}
