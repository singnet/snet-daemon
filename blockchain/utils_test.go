package blockchain

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesToBase64(t *testing.T) {
	base64 := BytesToBase64([]byte{1, 2, 254, 255})

	assert.Equal(t, "AQL+/w==", base64)
}
