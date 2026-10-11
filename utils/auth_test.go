package utils

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestVerifyAddress_Valid(t *testing.T) {
	addr := common.HexToAddress("0x7DF35C98f41F3AF0DF1DC4C7F7D4C19A71DD079F")
	addrLow := common.HexToAddress("0x7df35c98f41f3af0df1dc4c7f7d4c19a71Dd079f")

	err := VerifyAddress(addr, addrLow)
	require.NoError(t, err)
}

func TestVerifyAddress_Invalid(t *testing.T) {
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")

	err := VerifyAddress(addr1, addr2)
	require.Error(t, err)
}

func TestVerifySigner_Valid(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	message := []byte("test message")
	signature := GetSignature(message, privateKey)
	require.NotNil(t, signature)

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	err = VerifySigner(message, signature, address)
	require.NoError(t, err)
}

func TestVerifySigner_InvalidSignature(t *testing.T) {
	privateKey, _ := crypto.GenerateKey()
	message := []byte("test message")
	signature := GetSignature(message, privateKey)

	// incorrect signature
	signature[0] ^= 0x01

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	err := VerifySigner(message, signature, address)
	require.Error(t, err)
}

func TestVerifySigner_InvalidAddress(t *testing.T) {
	privateKey, _ := crypto.GenerateKey()
	message := []byte("test message")
	signature := GetSignature(message, privateKey)

	wrongAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	err := VerifySigner(message, signature, wrongAddr)
	require.Error(t, err)
}

func TestGetSignerAddressFromMessage_Valid(t *testing.T) {
	privateKey, _ := crypto.GenerateKey()
	message := []byte("another message")
	signature := GetSignature(message, privateKey)

	address, err := GetSignerAddressFromMessage(message, signature)
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(privateKey.PublicKey), *address)
}

func TestGetSignerAddressFromMessage_Invalid(t *testing.T) {
	message := []byte("message")
	invalidSignature := []byte("shortsig")

	address, err := GetSignerAddressFromMessage(message, invalidSignature)
	require.Error(t, err)
	require.Nil(t, address)
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
				Curve: crypto.S256(),
				D:     test.scalar,
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
