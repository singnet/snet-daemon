package utils

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSerializeDeserialize(t *testing.T) {
	original := map[string]int{"foo": 42, "bar": 7}

	serialized, err := Serialize(original)
	assert.NoError(t, err)

	var decoded map[string]int
	err = Deserialize(serialized, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestToChecksumAddress(t *testing.T) {
	lower := "0x52908400098527886E0F7030069857D2E4169EE7"
	checksumAddr := ToChecksumAddress(lower)
	assert.Equal(t, common.HexToAddress(lower), checksumAddr)
}

func TestParsePrivateKey(t *testing.T) {
	validKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	privBytes := crypto.FromECDSA(validKey)
	privHex := common.Bytes2Hex(privBytes)

	parsedKey := ParsePrivateKey(privHex)
	assert.NotNil(t, parsedKey)

	// Invalid key
	parsedKeyInvalid := ParsePrivateKey("not-a-valid-key")
	assert.Nil(t, parsedKeyInvalid)

	// Empty string returns nil
	parsedKeyEmpty := ParsePrivateKey("")
	assert.Nil(t, parsedKeyEmpty)
}

func TestGetAddressFromPrivateKeyECDSA(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.NoError(t, err)

	addr := GetAddressFromPrivateKeyECDSA(key)
	expected := crypto.PubkeyToAddress(key.PublicKey)
	assert.Equal(t, expected, addr)

	// Passing nil returns an empty address
	assert.Equal(t, common.Address{}, GetAddressFromPrivateKeyECDSA(nil))

	// Passing invalid public key type (simulate)
	badKey := &ecdsa.PrivateKey{} // no public key set
	assert.Equal(t, common.Address{}, GetAddressFromPrivateKeyECDSA(badKey))
}

func TestCheckIfHttps(t *testing.T) {
	assert.True(t, CheckIfHttps([]string{"https://example.com", "http://foo"}))
	assert.False(t, CheckIfHttps([]string{"http://example.com", "ws://foo"}))
	assert.False(t, CheckIfHttps([]string{}))
}

func TestIsJWT(t *testing.T) {
	validJWT := "header.payload.signature"
	invalidJWTs := []string{
		"",
		"only.onepart",
		"part1.part2.",
		".part2.part3",
		"part1..part3",
	}

	assert.True(t, IsJWT(validJWT))
	for _, invalid := range invalidJWTs {
		assert.False(t, IsJWT(invalid))
	}
}
