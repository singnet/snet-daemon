package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
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
