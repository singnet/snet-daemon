package utils

import (
	"bytes"
	"encoding/gob"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/authutils"
	"go.uber.org/zap"
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

func VerifySigner(message []byte, signature []byte, signer common.Address) error {
	derivedSigner, err := authutils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		zap.L().Error(err.Error())
		return err
	}
	if err = authutils.VerifyAddress(*derivedSigner, signer); err != nil {
		return err
	}
	return nil
}

func ToChecksumAddress(hexAddress string) common.Address {
	address := common.HexToAddress(hexAddress)
	mixedAddress := common.NewMixedcaseAddress(address)
	return mixedAddress.Address()
}

func CheckIfHttps(endpoints []string) bool {
	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "https") {
			return true
		}
	}
	return false
}
