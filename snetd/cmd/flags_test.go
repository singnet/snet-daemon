package cmd

import (
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = old

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestGenerateEvmKeys(t *testing.T) {
	out := captureStdout(t, func() {
		require.NoError(t, GenerateEvmKeys.RunE(GenerateEvmKeys, nil))
	})

	privateKeyMatch := regexp.MustCompile(`Private Key: ([0-9a-fA-F]{64})`).FindStringSubmatch(out)
	addressMatch := regexp.MustCompile(`Address: (0x[0-9a-fA-F]{40})`).FindStringSubmatch(out)

	require.Len(t, privateKeyMatch, 2, "output should contain a 64-hex-char private key")
	require.Len(t, addressMatch, 2, "output should contain a 0x-prefixed address")

	parsedKey := utils.ParsePrivateKey(privateKeyMatch[1])
	require.NotNil(t, parsedKey)
	assert.Equal(t, addressMatch[1], utils.GetAddressFromPrivateKeyECDSA(parsedKey).Hex())

	assert.Contains(t, out, "Save these keys or add them to the daemon config")
}
