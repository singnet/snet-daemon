package token

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/singnet/snet-daemon/config"
	"strings"
	"time"
)

type customJWTokenClaimsImpl struct {
	getGroupId func() string
}

func (service customJWTokenClaimsImpl) CreateToken(payLoad PayLoad) (CustomToken, error) {
	atClaims := jwt.MapClaims{}
	atClaims["payload"] = fmt.Sprintf("%v", payLoad)
	atClaims["orgId"] = config.GetString(config.OrganizationId)
	atClaims["groupId"] = service.getGroupId()
	//set the Expiry of the Token generated
	atClaims["exp"] = time.Now().UTC().
		Add(time.Second * time.Duration(config.GetInt(config.TokenExpiryInSeconds))).Unix()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	return jwtToken.SignedString([]byte(config.GetString(config.TokenSecretKey)))
}

func (service customJWTokenClaimsImpl) VerifyToken(receivedToken CustomToken, payLoad PayLoad) (err error) {
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

func (service customJWTokenClaimsImpl) checkJwtTokenClaims(claims jwt.MapClaims, payload PayLoad) (err error) {
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
