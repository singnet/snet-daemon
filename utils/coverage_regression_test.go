package utils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

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

func TestSerializeReturnsErrorForUnsupportedValue(t *testing.T) {
	serialized, err := Serialize(make(chan int))

	require.Error(t, err)
	require.Empty(t, serialized)
}

func TestGetSignatureReturnsNilWithoutPrivateKey(t *testing.T) {
	require.Nil(t, GetSignature([]byte("message"), nil))
}

func TestVerifySignerRejectsMalformedSignature(t *testing.T) {
	err := VerifySigner([]byte("message"), []byte("short"), common.Address{})

	require.EqualError(t, err, "incorrect signature length")
}

func TestGetSignerAddressRejectsInvalidSignatureData(t *testing.T) {
	signer, err := GetSignerAddressFromMessage([]byte("message"), make([]byte, 65))

	require.EqualError(t, err, "incorrect signature data")
	require.Nil(t, signer)
}

func TestGetSignatureLogsFatalForInvalidPrivateKey(t *testing.T) {
	for _, test := range []struct {
		name   string
		scalar *big.Int
	}{
		{name: "zero", scalar: new(big.Int)},
		{name: "curve order", scalar: new(big.Int).Set(crypto.S256().Params().N)},
	} {
		t.Run(test.name, func(t *testing.T) {
			core, logs := observer.New(zap.FatalLevel)
			// Exercise the fatal path without exiting the test process. These tests
			// must remain sequential because GetSignature uses the global logger.
			logger := zap.New(core, zap.WithFatalHook(zapcore.WriteThenPanic))
			t.Cleanup(zap.ReplaceGlobals(logger))
			key := &ecdsa.PrivateKey{
				PublicKey: ecdsa.PublicKey{Curve: crypto.S256()},
				D:         test.scalar,
			}

			require.Panics(t, func() { GetSignature([]byte("message"), key) })
			entries := logs.All()
			require.Len(t, entries, 1, "failure must be logged, not panic inside the signer")
			require.Equal(t, zap.FatalLevel, entries[0].Level)
			require.Contains(t, entries[0].Message, "Cannot sign test message:")
			require.Contains(t, entries[0].Message, "invalid private key")
		})
	}
}

func TestGetSignerAddressAcceptsRecoveryIDFormatsWithoutMutatingSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	message := []byte("message")
	signature := GetSignature(message, key)
	require.Len(t, signature, 65)
	require.LessOrEqual(t, signature[64], byte(1))

	for _, test := range []struct {
		name   string
		offset byte
	}{
		{name: "0 or 1", offset: 0},
		{name: "27 or 28", offset: 27},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded := bytes.Clone(signature)
			encoded[64] += test.offset
			original := bytes.Clone(encoded)
			address, err := GetSignerAddressFromMessage(message, encoded)
			require.NoError(t, err)
			require.NotNil(t, address)
			require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), *address)
			require.Equal(t, original, encoded)
		})
	}
}

func TestVerifySignerRejectsChangedMessage(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signature := GetSignature([]byte("original message"), key)
	signer := crypto.PubkeyToAddress(key.PublicKey)

	require.NoError(t, VerifySigner([]byte("original message"), signature, signer))
	require.Error(t, VerifySigner([]byte("changed message"), signature, signer))
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
