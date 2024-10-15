package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesToBase64(t *testing.T) {
	base64 := BytesToBase64([]byte{1, 2, 254, 255})
	assert.Equal(t, "AQL+/w==", base64)
}

func TestConvertBase64Encoding(t *testing.T) {
	if _, err := ConvertBase64Encoding("n@@###zNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U"); err != nil {
		assert.Equal(t, err.Error(), "illegal base64 data at input byte 1")
	}
}

func TestToChecksumAddress(t *testing.T) {
	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddress("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddress("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
}
