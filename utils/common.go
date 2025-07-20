package utils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
	"strings"

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

func ToChecksumAddress(hexAddress string) common.Address {
	address := common.HexToAddress(hexAddress)
	mixedAddress := common.NewMixedcaseAddress(address)
	return mixedAddress.Address()
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
	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
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
