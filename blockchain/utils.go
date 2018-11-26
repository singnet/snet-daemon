package blockchain

import (
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	"regexp"
	"strings"
)

func derivePrivateKey(mnemonic string, path ...uint32) (*ecdsa.PrivateKey, error) {
	seed := bip39.NewSeed(mnemonic, "")
	curr, err := hdkeychain.NewMaster(seed, &chaincfg.Params{})
	if err != nil {
		return nil, err
	}
	for i, childIndex := range path {
		if i < 3 {
			childIndex += hdkeychain.HardenedKeyStart
		}
		curr, err = curr.Child(childIndex)
		if err != nil {
			return nil, err
		}
	}
	privKey, err := curr.ECPrivKey()
	if err != nil {
		return nil, err
	}
	return privKey.ToECDSA(), nil
}

// ParseSignature parses Ethereum signature.
func ParseSignature(jobSignatureBytes []byte) (uint8, [32]byte, [32]byte, error) {
	r := [32]byte{}
	s := [32]byte{}

	if len(jobSignatureBytes) != 65 {
		return 0, r, s, fmt.Errorf("job signature incorrect length")
	}

	v := uint8(jobSignatureBytes[64])%27 + 27
	copy(r[:], jobSignatureBytes[0:32])
	copy(s[:], jobSignatureBytes[32:64])

	return v, r, s, nil
}

// AddressToHex converts Ethereum address to hex string representation.
func AddressToHex(address *common.Address) string {
	return address.Hex()
}

// BytesToBase64 converts array of bytes to base64 string.
func BytesToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// HexToBytes converts hex string to bytes array.
func HexToBytes(str string) []byte {
	return common.FromHex(str)
}

// HexToAddress converts hex string to Ethreum address.
func HexToAddress(str string) common.Address {
	return common.Address(common.BytesToAddress(HexToBytes(str)))
}

func StringToBytes32(str string) [32]byte {

	var byte32 [32]byte
	copy(byte32[:], []byte(str))
	return byte32
}

func RemoveSpecialCharactersfromHash(pString string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9=]")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(pString, "")
}

func convertBase64Encoding(groupID string) [32]byte {
	data, err := base64.StdEncoding.DecodeString(groupID)
	if err != nil {
		log.WithError(err).WithField("groupID", groupID).Panic("Error trying to base64.StdEncoding.DecodeString")
	}
	var byte32 [32]byte
	copy(byte32[:], data[:])
	return byte32
}

func FormatHash(ipfsHash string) string {
	log.WithField("metadataHash", ipfsHash).Debug("Before Formatting")
	ipfsHash = RemoveSpecialCharactersfromHash(ipfsHash)
	ipfsHash = strings.Replace(ipfsHash, IpfsPrefix, "", -1)
	log.WithField("metadataUri", ipfsHash).Debug("After Formatting")
	return ipfsHash
}
