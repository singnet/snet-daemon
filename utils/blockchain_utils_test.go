package utils

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func TestToChecksumAddressStr(t *testing.T) {
	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddressStr("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
	assert.Equal(t, "0xE9D09A6C296ACDd4C01b21F407aC93FDfC63e78c", ToChecksumAddressStr("0xe9d09A6C296aCdd4c01b21f407ac93fdfC63E78C"))
}

func TestToChecksumAddress(t *testing.T) {
	lower := "0x52908400098527886E0F7030069857D2E4169EE7"
	checksumAddr := ToChecksumAddress(lower)
	assert.Equal(t, common.HexToAddress(lower), checksumAddr)
}

func TestParseSignature(t *testing.T) {
	t.Run("valid signature", func(t *testing.T) {
		sig := make([]byte, 65)
		for i := 0; i < 65; i++ {
			sig[i] = byte(i + 1)
		}

		v, r, s, err := ParseSignature(sig)
		assert.NoError(t, err)

		expectedV := (sig[64] % 27) + 27
		assert.Equal(t, expectedV, v)
		assert.True(t, bytes.Equal(r[:], sig[0:32]))
		assert.True(t, bytes.Equal(s[:], sig[32:64]))
	})

	t.Run("too short signature", func(t *testing.T) {
		sig := make([]byte, 10)
		_, _, _, err := ParseSignature(sig)
		assert.Error(t, err)
		assert.Equal(t, "job signature incorrect length", err.Error())
	})

	t.Run("too long signature", func(t *testing.T) {
		sig := make([]byte, 70)
		_, _, _, err := ParseSignature(sig)
		assert.Error(t, err)
		assert.Equal(t, "job signature incorrect length", err.Error())
	})

	t.Run("v calculation", func(t *testing.T) {
		sig := make([]byte, 65)
		sig[64] = 28
		v, _, _, err := ParseSignature(sig)
		assert.NoError(t, err)

		expectedV := (sig[64] % 27) + 27
		assert.Equal(t, expectedV, v)
	})
}

func TestMakeTopicFilterer(t *testing.T) {
	t.Run("string shorter than 32 bytes", func(t *testing.T) {
		input := "short"
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.True(t, bytes.Equal(filter[0][:len(input)], []byte(input)))
		assert.Equal(t, byte(0), filter[0][len(input)]) // remaining bytes are zero
	})

	t.Run("string exactly 32 bytes", func(t *testing.T) {
		input := "12345678901234567890123456789012"
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.True(t, bytes.Equal(filter[0][:], []byte(input)))
	})

	t.Run("string longer than 32 bytes", func(t *testing.T) {
		input := "abcdefghijklmnopqrstuvwxyz1234567890"
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.True(t, bytes.Equal(filter[0][:], []byte(input)[:32]))
	})

	t.Run("empty string", func(t *testing.T) {
		input := ""
		filter := MakeTopicFilterer(input)
		assert.Len(t, filter, 1)
		assert.Equal(t, [32]byte{}, filter[0])
	})
}
