package utils

import (
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

// ParseSignature parses Ethereum signature.
func ParseSignature(jobSignatureBytes []byte) (uint8, [32]byte, [32]byte, error) {
	r := [32]byte{}
	s := [32]byte{}

	if len(jobSignatureBytes) != 65 {
		return 0, r, s, fmt.Errorf("job signature incorrect length")
	}

	v := (jobSignatureBytes[64])%27 + 27
	copy(r[:], jobSignatureBytes[0:32])
	copy(s[:], jobSignatureBytes[32:64])

	return v, r, s, nil
}

// AddressToHex converts an Ethereum address to hex string representation.
func AddressToHex(address *common.Address) string {
	return address.Hex()
}

// BytesToBase64 converts an array of bytes to base64 string.
func BytesToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// HexToBytes converts hex string to a byte array.
func HexToBytes(str string) []byte {
	return common.FromHex(str)
}

// HexToAddress converts hex string to Ethereum address.
func HexToAddress(str string) common.Address {
	return common.BytesToAddress(HexToBytes(str))
}

func StringToBytes32(str string) [32]byte {
	var byte32 [32]byte
	copy(byte32[:], str)
	return byte32
}

func ConvertBase64Encoding(str string) ([32]byte, error) {
	var byte32 [32]byte
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("String Passed", str))
		return byte32, err
	}
	copy(byte32[:], data[:])
	return byte32, nil
}

func ToChecksumAddressStr(hexAddress string) string {
	address := common.HexToAddress(hexAddress)
	mixedAddress := common.NewMixedcaseAddress(address)
	return mixedAddress.Address().String()
}

func ToChecksumAddress(hexAddress string) common.Address {
	address := common.HexToAddress(hexAddress)
	mixedAddress := common.NewMixedcaseAddress(address)
	return mixedAddress.Address()
}

/*
MakeTopicFilterer is used to generate a filter for querying Ethereum logs or contract events.
Ethereum topics (such as for events) are 32-byte fixed-size values (common for hashing
in Ethereum logs). This function takes a string parameter, converts it into a 32-byte array,
and returns it in a slice. This allows developers to create filters when looking for
specific events or log entries based on the topic.
*/
func MakeTopicFilterer(param string) [][32]byte {
	// Create a 32-byte array
	var param32Byte [32]byte

	// Convert the string to a byte slice and copy up to 32 bytes
	copy(param32Byte[:], []byte(param)[:min(len(param), 32)])

	// Return the filter with a single element (the 32-byte array)
	return [][32]byte{param32Byte}
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
	publicKeyECDSA := &privateKeyECDSA.PublicKey
	if publicKeyECDSA.X == nil || publicKeyECDSA.Y == nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

// GenerateKeys generates a new Ethereum key pair and returns the private key
// as a hex string (without the "0x" prefix) together with the derived address.
func GenerateKeys() (privateKeyHex string, address common.Address, err error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", common.Address{}, err
	}
	privateKeyHex = common.Bytes2Hex(crypto.FromECDSA(privateKey))
	address = GetAddressFromPrivateKeyECDSA(privateKey)
	return privateKeyHex, address, nil
}
