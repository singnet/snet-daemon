package utils

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsURLValid(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "valid http URL",
			endpoint: "http://example.com",
			expected: true,
		},
		{
			name:     "valid https URL",
			endpoint: "https://example.com/path",
			expected: true,
		},
		{
			name:     "missing scheme",
			endpoint: "example.com",
			expected: false,
		},
		{
			name:     "unsupported scheme ftp",
			endpoint: "ftp://example.com",
			expected: false,
		},
		{
			name:     "custom scheme h",
			endpoint: "h://example.com",
			expected: false,
		},
		{
			name:     "empty host",
			endpoint: "http:///path-only",
			expected: false,
		},
		{
			name:     "invalid URL with spaces",
			endpoint: "http://exa mple.com",
			expected: false,
		},
		{
			name:     "valid host with port",
			endpoint: "http://example.com:8080",
			expected: true,
		},
		{
			name:     "https host with query params",
			endpoint: "https://example.com/search?q=test",
			expected: true,
		},
		{
			name:     "uppercase scheme",
			endpoint: "HTTPS://example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsURLValid(tt.endpoint)
			assert.Equal(t, tt.expected, result, "endpoint: %s", tt.endpoint)
		})
	}
}

func TestSerializeReturnsErrorForUnsupportedValue(t *testing.T) {
	serialized, err := Serialize(make(chan int))

	require.Error(t, err)
	require.Empty(t, serialized)
}

func TestGetAddressFromPrivateKeyRejectsMissingCoordinate(t *testing.T) {
	for _, test := range []struct {
		name   string
		public ecdsa.PublicKey
	}{
		{name: "missing X", public: ecdsa.PublicKey{Curve: crypto.S256(), Y: big.NewInt(1)}},
		{name: "missing Y", public: ecdsa.PublicKey{Curve: crypto.S256(), X: big.NewInt(1)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			key := &ecdsa.PrivateKey{PublicKey: test.public}
			require.Equal(t, common.Address{}, GetAddressFromPrivateKeyECDSA(key))
		})
	}
}

func TestDeserializeRejectsMalformedDataAndInvalidDestination(t *testing.T) {
	serialized, err := Serialize("value")
	require.NoError(t, err)
	for _, test := range []struct {
		name        string
		data        string
		destination any
	}{
		{name: "empty data", destination: new(string)},
		{name: "malformed data", data: "not gob", destination: new(string)},
		{name: "truncated data", data: serialized[:len(serialized)-1], destination: new(string)},
		{name: "wrong type", data: serialized, destination: new(int)},
		{name: "non-pointer destination", data: serialized, destination: "value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, Deserialize(test.data, test.destination))
		})
	}
}
