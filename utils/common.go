package utils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
)

func Serialize(value any) (slice string, err error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	err = e.Encode(value)
	if err != nil {
		return
	}

	slice = b.String()
	return
}

func Deserialize(slice string, value any) (err error) {
	b := bytes.NewBuffer([]byte(slice))
	d := gob.NewDecoder(b)
	return d.Decode(value)
}

func ParsePrivateKey(privateKeyString string) (privateKey *ecdsa.PrivateKey) {
	if privateKeyString != "" {
		privateKey, err := crypto.HexToECDSA(privateKeyString)
		if err != nil {
			zap.L().Debug("Error parsing private key", zap.String("privateKeyString", privateKeyString), zap.Error(err))
			return nil
		}
		return privateKey
	}

	return nil
}

func GetAddressFromPrivateKeyECDSA(privateKeyECDSA *ecdsa.PrivateKey) common.Address {
	if privateKeyECDSA == nil {
		return common.Address{}
	}
	publicKey := privateKeyECDSA.Public()
	if publicKey == nil {
		return common.Address{}
	}
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok || publicKeyECDSA == nil {
		return common.Address{}
	}
	if publicKeyECDSA.X == nil || publicKeyECDSA.Y == nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

func CheckIfHttps(endpoints []string) bool {
	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "https") {
			return true
		}
	}
	return false
}

func IsJWT(token string) bool {
	parts := strings.Split(token, ".")
	// jwt always has 3 parts: header, payload, signature
	if len(parts) != 3 {
		return false
	}
	// check if each part is not empty
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}
	return true
}

func IsURLValid(endpoint string) bool {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
