package handler

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestGetBytesFromHexString(t *testing.T) {
	md := metadata.Pairs("test-key", "0xfFfE0100")

	bytes, err := GetBytesFromHex(md, "test-key")

	assert.Nil(t, err)
	assert.Equal(t, []byte{255, 254, 1, 0}, bytes)
}

func TestGetBytesFromHexStringNoPrefix(t *testing.T) {
	md := metadata.Pairs("test-key", "fFfE0100")

	bytes, err := GetBytesFromHex(md, "test-key")

	assert.Nil(t, err)
	assert.Equal(t, []byte{255, 254, 1, 0}, bytes)
}

func TestGetBytesFromHexStringNoValue(t *testing.T) {
	md := metadata.Pairs("unknown-key", "fFfE0100")

	_, err := GetBytesFromHex(md, "test-key")

	assert.Equal(t, NewGrpcErrorf(codes.InvalidArgument, "missing \"test-key\""), err)
}

func TestGetBytesFromHexStringTooManyValues(t *testing.T) {
	md := metadata.Pairs("test-key", "0x123", "test-key", "FED")

	_, err := GetBytesFromHex(md, "test-key")

	assert.Equal(t, NewGrpcErrorf(codes.InvalidArgument, "too many values for key \"test-key\": [0x123 FED]"), err)
}

func TestGetBigInt(t *testing.T) {
	md := metadata.Pairs("big-int-key", "12345")

	value, err := GetBigInt(md, "big-int-key")

	assert.Nil(t, err)
	assert.Equal(t, big.NewInt(12345), value)
}

func TestGetBigIntIncorrectValue(t *testing.T) {
	md := metadata.Pairs("big-int-key", "12345abc")

	_, err := GetBigInt(md, "big-int-key")

	assert.Equal(t, NewGrpcErrorf(codes.InvalidArgument, "incorrect format \"big-int-key\": \"12345abc\""), err)
}

func TestGetBigIntNoValue(t *testing.T) {
	md := metadata.Pairs()

	_, err := GetBigInt(md, "big-int-key")

	assert.Equal(t, NewGrpcErrorf(codes.InvalidArgument, "missing \"big-int-key\""), err)
}

func TestGetBigIntTooManyValues(t *testing.T) {
	md := metadata.Pairs("big-int-key", "12345", "big-int-key", "54321")

	_, err := GetBigInt(md, "big-int-key")

	assert.Equal(t, NewGrpcErrorf(codes.InvalidArgument, "too many values for key \"big-int-key\": [12345 54321]"), err)
}

func TestGetBytes(t *testing.T) {
	md := metadata.Pairs("binary-key-bin", string([]byte{0x00, 0x01, 0xFE, 0xFF}))

	value, err := GetBytes(md, "binary-key-bin")

	assert.Nil(t, err)
	assert.Equal(t, []byte{0, 1, 254, 255}, value)
}

func TestGetBytesIncorrectBinaryKey(t *testing.T) {
	md := metadata.Pairs("binary-key", string([]byte{0x00, 0x01, 0xFE, 0xFF}))

	_, err := GetBytes(md, "binary-key")

	assert.Equal(t, NewGrpcErrorf(codes.InvalidArgument, "incorrect binary key name \"binary-key\""), err)
}
