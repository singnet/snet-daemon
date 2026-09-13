package ipfsutils

import (
	"net/http"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile_FilecoinPrefix(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("filecoin-data"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{config.LighthouseEndpoint: ts.URL + "/"})

	data, err := ReadFile("filecoin://somecid123")
	require.NoError(t, err)
	assert.Equal(t, "filecoin-data", string(data))
}

func TestReadFile_IpfsPrefix(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("ipfs-data"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := ReadFile("ipfs://QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	require.NoError(t, err)
	assert.Equal(t, "ipfs-data", string(data))
}

// ========== ipfsutils.go ==========

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
