package utils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestBytesToBase64(t *testing.T) {
	base64 := BytesToBase64([]byte{1, 2, 254, 255})
	assert.Equal(t, "AQL+/w==", base64)
}

func TestConvertBase64Encoding(t *testing.T) {
	_, err := ConvertBase64Encoding("n@@###zNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U")
	assert.EqualError(t, err, "illegal base64 data at input byte 1")
}

func TestToChecksumAddressStr(t *testing.T) {
	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddressStr("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddressStr("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
}

func TestToChecksumAddress(t *testing.T) {
	lower := "0x52908400098527886E0F7030069857D2E4169EE7"
	checksumAddr := ToChecksumAddress(lower)
	assert.Equal(t, common.HexToAddress(lower), checksumAddr)
}

func TestParseSignature(t *testing.T) {
	t.Run("valid signature", func(t *testing.T) {
		sig := make([]byte, 65)
		for i := range 65 {
			sig[i] = byte(i + 1)
		}

		v, r, s, err := ParseSignature(sig)
		assert.NoError(t, err)

		expectedV := (sig[64] % 27) + 27
		assert.Equal(t, expectedV, v)
		assert.True(t, bytes.Equal(r[:], sig[0:32]))
		assert.True(t, bytes.Equal(s[:], sig[32:64]))
	})

	t.Run("too short signature", func(t *testing.T) {
		sig := make([]byte, 10)
		_, _, _, err := ParseSignature(sig)
		assert.Error(t, err)
		assert.Equal(t, "job signature incorrect length", err.Error())
	})

	t.Run("too long signature", func(t *testing.T) {
		sig := make([]byte, 70)
		_, _, _, err := ParseSignature(sig)
		assert.Error(t, err)
		assert.Equal(t, "job signature incorrect length", err.Error())
	})

	t.Run("v calculation", func(t *testing.T) {
		sig := make([]byte, 65)
		sig[64] = 28
		v, _, _, err := ParseSignature(sig)
		assert.NoError(t, err)

		expectedV := (sig[64] % 27) + 27
		assert.Equal(t, expectedV, v)
	})
}

func TestMakeTopicFilterer(t *testing.T) {
	t.Run("string shorter than 32 bytes", func(t *testing.T) {
		input := "short"
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.True(t, bytes.Equal(filter[0][:len(input)], []byte(input)))
		assert.Equal(t, byte(0), filter[0][len(input)]) // remaining bytes are zero
	})

	t.Run("string exactly 32 bytes", func(t *testing.T) {
		input := "12345678901234567890123456789012"
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.True(t, bytes.Equal(filter[0][:], []byte(input)))
	})

	t.Run("string longer than 32 bytes", func(t *testing.T) {
		input := "abcdefghijklmnopqrstuvwxyz1234567890"
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.True(t, bytes.Equal(filter[0][:], []byte(input)[:32]))
	})

	t.Run("empty string", func(t *testing.T) {
		input := ""
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.Equal(t, [32]byte{}, filter[0])
	})
}

func TestBlockchainConversionHelpers(t *testing.T) {
	address := common.HexToAddress("0x00000000000000000000000000000000000000ab")
	require.Equal(t, address.Hex(), AddressToHex(&address))
	require.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, HexToBytes("0xdeadbeef"))
	require.Equal(t, address, HexToAddress(address.Hex()))

	expectedBytes32 := [32]byte{}
	copy(expectedBytes32[:], "group-id")
	require.Equal(t, expectedBytes32, StringToBytes32("group-id"))

	raw := make([]byte, 32)
	for index := range raw {
		raw[index] = byte(index)
	}
	decoded, err := ConvertBase64Encoding(base64.StdEncoding.EncodeToString(raw))
	require.NoError(t, err)
	require.Equal(t, [32]byte(raw), decoded)
}

func TestParsePrivateKey(t *testing.T) {
	keyHex := strings.Repeat("0123456789abcdef", 4)
	wantKey, err := crypto.HexToECDSA(keyHex)
	require.NoError(t, err)

	for _, tc := range []struct {
		name  string
		input string
		valid bool
	}{
		{"unprefixed", keyHex, true},
		{"lowercase prefix", "0x" + keyHex, true},
		{"uppercase prefix", "0X" + keyHex, true},
		{"uppercase hex", "0x" + strings.ToUpper(keyHex), true},
		{"empty", "", false},
		{"prefix only", "0x", false},
		{"uppercase prefix only", "0X", false},
		{"invalid hex", "0x" + strings.Repeat("g", 64), false},
		{"short key", keyHex[:62], false},
		{"long key", "0x" + keyHex + "00", false},
		{"odd length", "0x" + keyHex[:63], false},
		{"repeated prefix", "0x0x" + keyHex, false},
		{"mixed repeated prefix", "0x0X" + keyHex, false},
		{"leading whitespace", " 0x" + keyHex, false},
		{"trailing whitespace", keyHex + " ", false},
		{"zero scalar", "0x" + strings.Repeat("0", 64), false},
		{"curve order", "0x" + crypto.S256().Params().N.Text(16), false},
		{"above curve order", strings.Repeat("f", 64), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParsePrivateKeyWithError(tc.input)
			key := ParsePrivateKey(tc.input)
			if !tc.valid {
				require.Error(t, err)
				require.Nil(t, parsed)
				require.Nil(t, key)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, parsed)
			require.Equal(t, crypto.FromECDSA(wantKey), crypto.FromECDSA(parsed))
			require.NotNil(t, key)
			require.Equal(t, crypto.FromECDSA(wantKey), crypto.FromECDSA(key))
			require.Equal(t, crypto.PubkeyToAddress(wantKey.PublicKey), crypto.PubkeyToAddress(key.PublicKey))
		})
	}
}

func TestParsePrivateKeyDoesNotLogInput(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	t.Cleanup(zap.ReplaceGlobals(zap.New(core)))
	invalidKey := "0x" + strings.Repeat("1", 63) + "g"
	require.Nil(t, ParsePrivateKey(invalidKey))

	entries := logs.All()
	require.Len(t, entries, 1)
	require.NotContains(t, entries[0].Message, invalidKey)
	fields := entries[0].ContextMap()
	require.NotContains(t, fields, "privateKeyString")
	encodedFields, err := json.Marshal(fields)
	require.NoError(t, err)
	require.NotContains(t, string(encodedFields), invalidKey)
}

func TestGetAddressFromPrivateKeyECDSA(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.NoError(t, err)

	addr := GetAddressFromPrivateKeyECDSA(key)
	expected := crypto.PubkeyToAddress(key.PublicKey)
	assert.Equal(t, expected, addr)

	// Passing nil returns an empty address
	assert.Equal(t, common.Address{}, GetAddressFromPrivateKeyECDSA(nil))

	// An uninitialized public key cannot be encoded as a secp256k1 public key.
	badKey := &ecdsa.PrivateKey{} // no public key set
	assert.Equal(t, common.Address{}, GetAddressFromPrivateKeyECDSA(badKey))
}

func TestGetAddressFromPrivateKeySecp256k1Vector(t *testing.T) {
	key, err := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	require.NoError(t, err)
	expected := common.HexToAddress("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf")
	for _, tc := range []struct {
		name string
		key  *ecdsa.PrivateKey
	}{
		{"private key", key},
		{"public key only", &ecdsa.PrivateKey{PublicKey: key.PublicKey}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, expected, GetAddressFromPrivateKeyECDSA(tc.key))
		})
	}
}

func TestGetAddressFromPrivateKeyRejectsMissingCoordinate(t *testing.T) {
	for _, test := range []struct {
		name   string
		public ecdsa.PublicKey
	}{
		{name: "missing both", public: ecdsa.PublicKey{Curve: crypto.S256()}},
		{name: "missing X", public: ecdsa.PublicKey{Curve: crypto.S256(), Y: big.NewInt(1)}},
		{name: "missing Y", public: ecdsa.PublicKey{Curve: crypto.S256(), X: big.NewInt(1)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			key := &ecdsa.PrivateKey{PublicKey: test.public}
			require.Equal(t, common.Address{}, GetAddressFromPrivateKeyECDSA(key))
		})
	}
}

func TestGenerateKeys(t *testing.T) {
	privateKeyHex, address, err := GenerateKeys()
	require.NoError(t, err)

	require.Len(t, privateKeyHex, 64)

	parsedKey := ParsePrivateKey(privateKeyHex)
	require.NotNil(t, parsedKey)
	assert.Equal(t, address, GetAddressFromPrivateKeyECDSA(parsedKey))
	assert.Equal(t, crypto.PubkeyToAddress(parsedKey.PublicKey), address)
}
