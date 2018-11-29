package blockchain

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBytesToBase64(t *testing.T) {
	base64 := BytesToBase64([]byte{1, 2, 254, 255})
	assert.Equal(t, "AQL+/w==", base64)
}

func TestFormatHash(t *testing.T) {
	s2 := []byte("ipfs://Here is a string....+=")
	hash := FormatHash(string(s2))
	assert.Equal(t, hash, "Hereisastring=")
	s2 = []byte("QmaGnQ3iVZPuPwdam2rEeQcCSoCYRpxjnZhQ6Z2oeeRSrp")

	b4 := append(s2, make([]byte, 3)...)
	assert.NotEqual(t, "QmaGnQ3iVZPuPwdam2rEeQcCSoCYRpxjnZhQ6Z2oeeRSrp", string(b4))
	assert.Equal(t, "QmaGnQ3iVZPuPwdam2rEeQcCSoCYRpxjnZhQ6Z2oeeRSrp", FormatHash(string(b4)))
}
func TestConvertBase64Encoding(t *testing.T) {

	_, err := ConvertBase64Encoding("n@@###zNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U")
	assert.Equal(t, err.Error(), "illegal base64 data at input byte 1")

}
