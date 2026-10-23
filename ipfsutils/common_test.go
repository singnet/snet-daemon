package ipfsutils

import (
	"context"
	"testing"
	"time"

	mb "github.com/multiformats/go-multibase"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileReferences(t *testing.T) {
	for _, lighthouse := range []bool{false, true} {
		prefix := IpfsPrefix
		if lighthouse {
			prefix = FilecoinPrefix
		}
		t.Run(prefix, func(t *testing.T) {
			block := unixFSFixture(t, []byte("reference fixture"))
			serveBlocks(t, lighthouse, block)
			file, err := ReadFile(" \t" + prefix + block.cid.String() + "\x00\x00\n")
			require.NoError(t, err)
			assert.Equal(t, "reference fixture", string(file))
		})
	}
	block := rawFixture(t, []byte("bare CID"))
	serveBlocks(t, false, block)
	file, err := ReadFile(block.cid.String())
	require.NoError(t, err)
	assert.Equal(t, "bare CID", string(file))
}

func TestParseReferenceMultibase(t *testing.T) {
	block := rawFixture(t, []byte("\xfb\xff\xff CID multibase fixture"))
	for _, encoding := range []mb.Encoding{mb.Base32, mb.Base58BTC, mb.Base64, mb.Base64pad, mb.Base64url, mb.Base64urlPad} {
		encoded, err := block.cid.StringOfBase(encoding)
		require.NoError(t, err)
		parsed, err := parseReference(IpfsPrefix+encoded, IpfsPrefix)
		require.NoError(t, err)
		assert.Equal(t, block.cid, parsed)
	}
}

func TestReadFileRejectsMalformedReferences(t *testing.T) {
	block := rawFixture(t, []byte("reference"))
	valid := block.cid.String()
	base64CID, err := block.cid.StringOfBase(mb.Base64)
	require.NoError(t, err)
	withTestConfig(t, map[string]string{config.IpfsEndpoint: ":invalid", config.LighthouseEndpoint: ":invalid"})
	for _, reference := range []string{
		"", "not-a-cid", "https://example.com/ipfs/" + valid, "/ipfs/" + valid,
		IpfsPrefix + valid + "/file", FilecoinPrefix + valid + "?download=true",
		IpfsPrefix + valid + "!", IpfsPrefix + "ipfs://" + valid,
		valid[:10] + " " + valid[10:], valid[:10] + "\x00" + valid[10:],
		base64CID[:10] + "\n" + base64CID[10:],
	} {
		t.Run(reference, func(t *testing.T) {
			file, err := ReadFile(reference)
			require.ErrorContains(t, err, "invalid content reference")
			assert.Nil(t, file)
		})
	}
}

func TestRetrievalTimeout(t *testing.T) {
	for _, tc := range []struct {
		configured string
		ipfs       time.Duration
		lighthouse time.Duration
	}{{"3", 3 * time.Second, 3 * time.Second}, {"0", 0, 30 * time.Second}} {
		t.Run(tc.configured, func(t *testing.T) {
			withTestConfig(t, map[string]string{config.IpfsTimeout: tc.configured})
			assert.Equal(t, tc.ipfs, retrievalTimeout(false))
			assert.Equal(t, tc.lighthouse, retrievalTimeout(true))
		})
	}
	ctx, cancel := retrievalContext(context.Background(), 0)
	defer cancel()
	_, deadline := ctx.Deadline()
	assert.False(t, deadline)
}
