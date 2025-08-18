package ipfsutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilecoinReadFile(t *testing.T) {
	file, err := ReadFile("filecoin://bafkreibk4ham7y6mwad2qxwyrhpmxh2fho7xwvsfw26lcra5bt5r5fvcwe")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	if err != nil && file != nil {
		t.Log(string(file))
	}
}

func TestIpfsReadFile(t *testing.T) {
	file, err := ReadFile("ipfs://QmQcT5SJB9s8LXom8zuNGksCa7d34XbVn52dACWvgzeWAW")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	if err != nil && file != nil {
		t.Log(string(file))
	}
}

func TestFormatHash(t *testing.T) {
	s2 := []byte("ipfs://Here is a string....+=")
	hash := formatHash(string(s2))
	assert.Equal(t, hash, "Hereisastring=")

	s2 = []byte("filecoin://QmaGnQ3iVZPuPwdam2rEeQcCSoCYRpxjnZhQ6Z2oeeRSrp")
	b4 := append(s2, make([]byte, 3)...)
	assert.NotEqual(t, "QmaGnQ3iVZPuPwdam2rEeQcCSoCYRpxjnZhQ6Z2oeeRSrp", string(b4))
	assert.Equal(t, "QmaGnQ3iVZPuPwdam2rEeQcCSoCYRpxjnZhQ6Z2oeeRSrp", formatHash(string(b4)))
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
			output := removeSpecialCharacters(tc.input)
			if output != tc.expectedOutput {
				t.Errorf("RemoveSpecialCharactersfromHash(%q) = %q; want %q", tc.input, output, tc.expectedOutput)
			}
		})
	}
}
