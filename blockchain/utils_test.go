package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	if _, err := ConvertBase64Encoding("n@@###zNEetD1kzU3PZqR4nHPS8erDkrUK0hN4iCBQ4vH5U"); err != nil {
		assert.Equal(t, err.Error(), "illegal base64 data at input byte 1")
	}

}

func TestToChecksumAddress(t *testing.T) {

	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddress("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))

	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddress("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
}

func TestRemoveSpecialCharactersfromHash(t *testing.T) {
	testCases := []struct {
		input          string
		expectedOutput string
	}{
		{"abc123", "abc123"},
		{"abc123!@#", "abc123"},
		{"a1b2c3 ~`!@#$%^&*()_+-={}[]|\\:;\"'<>,.?/", "a1b2c3="},
		{"abc=123", "abc=123"},
		{"a1!b2@c3#=4", "a1b2c3=4"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			output := RemoveSpecialCharactersfromHash(tc.input)
			if output != tc.expectedOutput {
				t.Errorf("RemoveSpecialCharactersfromHash(%q) = %q; want %q", tc.input, output, tc.expectedOutput)
			}
		})
	}
}
