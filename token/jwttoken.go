package token

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"strings"
	"time"
)

type customJWTokenServiceImpl struct {
	getGroupId func() string
}

// This will be used in components as a service to Create and Validate tokens
func NewJWTTokenService(data blockchain.OrganizationMetaData) Manager {
	return &customJWTokenServiceImpl{
		getGroupId: func() string {
			return data.GetGroupIdString()
		},
	}
}

func (service customJWTokenServiceImpl) CreateToken(payLoad PayLoad) (CustomToken, error) {
	atClaims := jwt.MapClaims{}
	atClaims["payload"] = fmt.Sprintf("%v", payLoad)
	atClaims["orgId"] = config.GetString(config.OrganizationId)
	atClaims["groupId"] = service.getGroupId()
	//set the Expiry of the Token generated
	atClaims["exp"] = time.Now().UTC().
		Add(time.Minute * time.Duration(config.GetInt(config.TokenExpiryInMinutes))).Unix()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	return jwtToken.SignedString([]byte(config.GetString(config.TokenSecretKey)))
}

func (service customJWTokenServiceImpl) VerifyToken(receivedToken CustomToken, payLoad PayLoad) (err error) {
	tokenString := fmt.Sprintf("%v", receivedToken)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method : %v", token.Header["alg"])
		}
		return []byte(config.GetString(config.TokenSecretKey)), nil
	})
	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if err = service.checkJwtTokenClaims(claims, payLoad); err != nil {
			return err
		}
	}
	return nil
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
