package utils

import (
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

// ParseSignature splits a 65-byte R || S || V signature and returns normalized
// V, R, and S. Conventional recovery IDs 0/1 and 27/28 become 27/28; only length
// is checked here, not ECDSA validity or the signer's identity.
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

// AddressToHex returns the checksummed hexadecimal form of an Ethereum address.
func AddressToHex(address *common.Address) string {
	return address.Hex()
}

// BytesToBase64 converts an array of bytes to base64 string.
func BytesToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// HexToBytes uses go-ethereum's permissive hex decoder without returning a
// decoding error. Callers needing strict input validation must check it separately.
func HexToBytes(str string) []byte {
	return common.FromHex(str)
}

// HexToAddress decodes hex and pads or truncates to a 20-byte address. It does not
// validate address length, hexadecimal syntax, or an input checksum.
func HexToAddress(str string) common.Address {
	return common.BytesToAddress(HexToBytes(str))
}

// StringToBytes32 copies UTF-8 bytes into a zero-padded 32-byte value, truncating
// longer input. Registry organization and service IDs use this representation;
// the string is not hashed.
func StringToBytes32(str string) [32]byte {
	var byte32 [32]byte
	copy(byte32[:], str)
	return byte32
}

// ConvertBase64Encoding decodes standard base64 into a 32-byte value, padding or
// truncating the result. It reports decoding errors but does not enforce length;
// organization metadata group IDs use this conversion.
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

// ToChecksumAddressStr parses an address and returns checksummed hex text without
// validating the original input's length or checksum.
func ToChecksumAddressStr(hexAddress string) string {
	address := common.HexToAddress(hexAddress)
	mixedAddress := common.NewMixedcaseAddress(address)
	return mixedAddress.Address().String()
}

// ToChecksumAddress parses an address value. A common.Address contains bytes, not
// a textual checksum; this helper does not validate the original address text.
func ToChecksumAddress(hexAddress string) common.Address {
	address := common.HexToAddress(hexAddress)
	mixedAddress := common.NewMixedcaseAddress(address)
	return mixedAddress.Address()
}

// MakeTopicFilterer builds a single indexed-bytes32 filter value by copying and
// zero-padding up to 32 UTF-8 bytes. Registry event filters use it for organization
// and service IDs. It does not hash an event signature or a dynamic string value.
func MakeTopicFilterer(param string) [][32]byte {
	var param32Byte [32]byte

	copy(param32Byte[:], []byte(param)[:min(len(param), 32)])

	return [][32]byte{param32Byte}
}

// ParsePrivateKeyWithError parses a 32-byte hexadecimal secp256k1 private key
// with an optional 0x or 0X prefix. All other validation is left to HexToECDSA.
func ParsePrivateKeyWithError(privateKeyString string) (*ecdsa.PrivateKey, error) {
	if strings.HasPrefix(privateKeyString, "0x") || strings.HasPrefix(privateKeyString, "0X") {
		privateKeyString = privateKeyString[2:]
	}
	return crypto.HexToECDSA(privateKeyString)
}

// ParsePrivateKey returns nil for an empty or invalid private key.
func ParsePrivateKey(privateKeyString string) (privateKey *ecdsa.PrivateKey) {
	if privateKeyString != "" {
		privateKey, err := ParsePrivateKeyWithError(privateKeyString)
		if err != nil {
			zap.L().Debug("Error parsing private key", zap.Error(err))
			return nil
		}
		return privateKey
	}

	return nil
}

// GetAddressFromPrivateKeyECDSA derives an Ethereum address, returning the zero
// address for a nil key or a public key that cannot be encoded.
func GetAddressFromPrivateKeyECDSA(privateKeyECDSA *ecdsa.PrivateKey) common.Address {
	if privateKeyECDSA == nil {
		return common.Address{}
	}
	publicKeyECDSA := &privateKeyECDSA.PublicKey
	// Use go-ethereum's nil-safe secp256k1 encoder; ecdsa.PublicKey.Bytes
	// only supports NIST curves.
	if len(crypto.FromECDSAPub(publicKeyECDSA)) == 0 {
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
