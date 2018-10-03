package blockchain

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestGetBytes(t *testing.T) {
	md := metadata.Pairs("test-key", "0xfFfE0100")

	bytes, err := getBytes(md, "test-key")

	assert.Nil(t, err)
	assert.Equal(t, []byte{255, 254, 1, 0}, bytes)
}

func TestGetBytesNoPrefix(t *testing.T) {
	md := metadata.Pairs("test-key", "fFfE0100")

	bytes, err := getBytes(md, "test-key")

	assert.Nil(t, err)
	assert.Equal(t, []byte{255, 254, 1, 0}, bytes)
}

func TestGetBytesNoValue(t *testing.T) {
	md := metadata.Pairs("unknown-key", "fFfE0100")

	bytes, err := getBytes(md, "test-key")

	assert.Equal(t, status.Errorf(codes.InvalidArgument, "missing \"test-key\""), err)
	assert.Nil(t, bytes)
}
