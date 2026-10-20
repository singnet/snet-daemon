package ipfsutils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/boxo/ipld/unixfs"
	ufspb "github.com/ipfs/boxo/ipld/unixfs/pb"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	mh "github.com/multiformats/go-multihash"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type blockFixture struct {
	cid  cid.Cid
	data []byte
	size uint64
}

func encodedFixture(t *testing.T, codec uint64, data []byte, size uint64) blockFixture {
	t.Helper()
	// Hash fixture bytes independently of the production CID.Prefix().Sum path.
	digest := sha256.Sum256(data)
	hash, err := mh.Encode(digest[:], mh.SHA2_256)
	require.NoError(t, err)
	c := cid.NewCidV1(codec, hash)
	if codec == cid.DagProtobuf {
		c = cid.NewCidV0(hash)
	}
	return blockFixture{cid: c, data: data, size: size}
}

func rawFixture(t *testing.T, data []byte) blockFixture {
	t.Helper()
	return encodedFixture(t, cid.Raw, data, uint64(len(data)))
}

func unixFSFixture(t *testing.T, inline []byte, children ...blockFixture) blockFixture {
	t.Helper()
	fs := unixfs.NewFSNode(unixfs.TFile)
	fs.SetData(inline)
	node := new(merkledag.ProtoNode)
	for _, child := range children {
		fs.AddBlockSize(child.size)
		require.NoError(t, node.AddRawLink("", &ipld.Link{Cid: child.cid, Size: uint64(len(child.data))}))
	}
	data, err := fs.GetBytes()
	require.NoError(t, err)
	node.SetData(data)
	return encodedFixture(t, cid.DagProtobuf, node.RawData(), fs.FileSize())
}

func serveBlocks(t *testing.T, lighthouse bool, blocks ...blockFixture) {
	t.Helper()
	lookup := make(map[cid.Cid][]byte)
	for _, block := range blocks {
		lookup[block.cid] = block.data
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Cookie"))
		assert.Equal(t, "application/vnd.ipld.raw", r.Header.Get("Accept"))
		var hash string
		if lighthouse {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.True(t, strings.HasPrefix(r.URL.Path, "/ipfs/"))
			assert.Equal(t, "raw", r.URL.Query().Get("format"))
			hash = strings.TrimPrefix(r.URL.Path, "/ipfs/")
		} else {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v0/block/get", r.URL.Path)
			hash = r.URL.Query().Get("arg")
		}
		c, err := cid.Decode(hash)
		if !assert.NoError(t, err) {
			http.Error(w, "invalid CID", http.StatusBadRequest)
			return
		}
		data, ok := lookup[c]
		if !ok {
			http.Error(w, "missing block", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.ipld.raw")
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	key, endpoint := config.IpfsEndpoint, server.URL
	if lighthouse {
		key, endpoint = config.LighthouseEndpoint, server.URL+"/ipfs/"
	}
	withTestConfig(t, map[string]string{key: endpoint})
}

func TestVerifiedUnixFSFixtures(t *testing.T) {
	// Published UnixFS "hello world\n" block, also used in go-unixfsnode fixtures.
	helloData, err := hex.DecodeString("0a120802120c68656c6c6f20776f726c640a180c")
	require.NoError(t, err)
	helloCID, err := cid.Decode("QmT78zSuBmuS4z925WZfrqQ1qHaJ56DQaTfyMUF7F8ff5o")
	require.NoError(t, err)
	hello := blockFixture{cid: helloCID, data: helloData, size: 12}
	a := rawFixture(t, []byte("A"))
	b := unixFSFixture(t, []byte("B"))
	nested := unixFSFixture(t, []byte("nested:"), b, a)
	root := unixFSFixture(t, []byte("prefix:"), a, nested, a)
	legacy := encodedFixture(t, cid.DagProtobuf, merkledag.NodeWithData(unixfs.WrapData([]byte("legacy"))).RawData(), 6)
	v1 := b
	v1.cid = cid.NewCidV1(cid.DagProtobuf, b.cid.Hash())
	largeLeaf := rawFixture(t, bytes.Repeat([]byte("x"), maxBlockSize/2+1))
	largeFile := unixFSFixture(t, nil, largeLeaf, largeLeaf)
	for _, tc := range []struct {
		name   string
		root   blockFixture
		blocks []blockFixture
		want   []byte
	}{
		{"known CIDv0", hello, nil, []byte("hello world\n")},
		{"raw", a, nil, []byte("A")},
		{"raw at block limit", rawFixture(t, bytes.Repeat([]byte("x"), maxBlockSize)), nil, bytes.Repeat([]byte("x"), maxBlockSize)},
		{"DAG-PB CIDv1", v1, nil, []byte("B")},
		{"legacy raw", legacy, nil, []byte("legacy")},
		{"empty", unixFSFixture(t, nil), nil, nil},
		{"inline and repeated nested links", root, []blockFixture{a, b, nested}, []byte("prefix:Anested:BAA")},
		{"file larger than block limit", largeFile, []blockFixture{largeLeaf}, bytes.Repeat([]byte("x"), maxBlockSize+2)},
	} {
		for _, lighthouse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/gateway=%t", tc.name, lighthouse), func(t *testing.T) {
				serveBlocks(t, lighthouse, append(tc.blocks, tc.root)...)
				prefix := IpfsPrefix
				if lighthouse {
					prefix = FilecoinPrefix
				}
				got, err := ReadFile(prefix + tc.root.cid.String())
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			})
		}
	}
}

func TestVerifiedMultihash(t *testing.T) {
	data := []byte("SHA-512 file")
	digest := sha512.Sum512(data)
	hash, err := mh.Encode(digest[:], mh.SHA2_512)
	require.NoError(t, err)
	block := blockFixture{cid: cid.NewCidV1(cid.Raw, hash), data: data}
	serveBlocks(t, false, block)
	got, err := GetIpfsFile(block.cid.String())
	require.NoError(t, err)
	assert.Equal(t, data, got)

	hash, err = mh.Encode(data, mh.IDENTITY)
	require.NoError(t, err)
	withTestConfig(t, map[string]string{config.IpfsEndpoint: "http://127.0.0.1:1"})
	got, err = GetIpfsFile(cid.NewCidV1(cid.Raw, hash).String())
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestVerifiedUnixFSFailures(t *testing.T) {
	child := rawFixture(t, []byte("child"))
	root := unixFSFixture(t, []byte("prefix"), child)
	corruptChild := child
	corruptChild.data = []byte("forged")
	corruptRoot := root
	corruptRoot.data = []byte("forged")
	badPB := encodedFixture(t, cid.DagProtobuf, []byte("invalid protobuf"), 0)
	badUnixFS := encodedFixture(t, cid.DagProtobuf, merkledag.NodeWithData([]byte{0xff}).RawData(), 0)
	directory := encodedFixture(t, cid.DagProtobuf, merkledag.NodeWithData(unixfs.FolderPBData()).RawData(), 0)
	symlinkData, err := unixfs.SymlinkData("target")
	require.NoError(t, err)
	symlink := encodedFixture(t, cid.DagProtobuf, merkledag.NodeWithData(symlinkData).RawData(), 0)
	badCount := malformedUnixFSFixture(t, &ufspb.Data{Type: new(ufspb.Data_File), Filesize: new(uint64(0))}, child)
	badSize := malformedUnixFSFixture(t, &ufspb.Data{Type: new(ufspb.Data_File), Data: []byte("x"), Filesize: new(uint64(2))})
	badChildSize := malformedUnixFSFixture(t, &ufspb.Data{Type: new(ufspb.Data_File), Filesize: new(uint64(3)), Blocksizes: []uint64{3}}, child)
	overflow := malformedUnixFSFixture(t, &ufspb.Data{Type: new(ufspb.Data_File), Data: []byte("x"), Filesize: new(uint64(0)), Blocksizes: []uint64{math.MaxUint64}}, child)
	for _, tc := range []struct {
		name   string
		root   blockFixture
		blocks []blockFixture
		want   string
	}{
		{"forged root", corruptRoot, nil, "CID mismatch"},
		{"forged child", root, []blockFixture{corruptChild}, "CID mismatch"},
		{"missing child", root, nil, "HTTP 404"},
		{"invalid DAG-PB", badPB, nil, "decode DAG-PB"},
		{"invalid UnixFS", badUnixFS, nil, "decode UnixFS"},
		{"directory", directory, nil, "unsupported UnixFS type"},
		{"symlink", symlink, nil, "unsupported UnixFS type"},
		{"link count", badCount, []blockFixture{child}, "inconsistent links"},
		{"filesize", badSize, nil, "filesize mismatch"},
		{"child size", badChildSize, []blockFixture{child}, "child size mismatch"},
		{"overflow", overflow, []blockFixture{child}, "size overflow"},
	} {
		for _, lighthouse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/gateway=%t", tc.name, lighthouse), func(t *testing.T) {
				serveBlocks(t, lighthouse, append(tc.blocks, tc.root)...)
				prefix := IpfsPrefix
				if lighthouse {
					prefix = FilecoinPrefix
				}
				data, err := ReadFile(prefix + tc.root.cid.String())
				require.ErrorContains(t, err, tc.want)
				assert.Nil(t, data)
			})
		}
	}
}

func malformedUnixFSFixture(t *testing.T, fs *ufspb.Data, children ...blockFixture) blockFixture {
	t.Helper()
	data, err := proto.Marshal(fs)
	require.NoError(t, err)
	node := merkledag.NodeWithData(data)
	for _, child := range children {
		require.NoError(t, node.AddRawLink("", &ipld.Link{Cid: child.cid}))
	}
	return encodedFixture(t, cid.DagProtobuf, node.RawData(), fs.GetFilesize())
}

func TestUnsupportedCID(t *testing.T) {
	shortHash, err := mh.Encode([]byte{0}, mh.SHA2_256)
	require.NoError(t, err)
	unknownHash, err := mh.Encode(make([]byte, 32), 0x1234)
	require.NoError(t, err)
	for _, c := range []cid.Cid{
		encodedFixture(t, cid.DagCBOR, []byte("unsupported"), 0).cid,
		cid.NewCidV1(cid.Raw, shortHash), cid.NewCidV1(cid.Raw, unknownHash),
	} {
		data, err := readVerifiedUnixFSFile(context.Background(), c, func(context.Context, cid.Cid) ([]byte, error) {
			t.Fatal("unsupported CID must be rejected before a network request")
			return nil, nil
		})
		require.Error(t, err)
		assert.Nil(t, data)
	}
}

func TestReadFileCancellation(t *testing.T) {
	for _, lighthouse := range []bool{false, true} {
		t.Run(fmt.Sprint(lighthouse), func(t *testing.T) {
			child := rawFixture(t, []byte("child"))
			root := unixFSFixture(t, []byte("prefix"), child)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.String(), root.cid.String()) {
					_, _ = w.Write(root.data)
					return
				}
				cancel()
				<-r.Context().Done()
			}))
			defer server.Close()
			key, endpoint, prefix := config.IpfsEndpoint, server.URL, IpfsPrefix
			if lighthouse {
				key, endpoint, prefix = config.LighthouseEndpoint, server.URL+"/ipfs/", FilecoinPrefix
			}
			withTestConfig(t, map[string]string{key: endpoint})
			data, err := ReadFileContext(ctx, prefix+root.cid.String())
			require.ErrorIs(t, err, context.Canceled)
			assert.Nil(t, data)
		})
	}
}

func TestReadFileDeadline(t *testing.T) {
	block := rawFixture(t, []byte("delayed"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	withTestConfig(t, map[string]string{config.IpfsEndpoint: server.URL, config.LighthouseEndpoint: server.URL + "/ipfs/"})
	for _, prefix := range []string{IpfsPrefix, FilecoinPrefix} {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		data, err := ReadFileContext(ctx, prefix+block.cid.String())
		cancel()
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Nil(t, data)
	}
}

func TestReadFileOversizedBlock(t *testing.T) {
	block := rawFixture(t, bytes.Repeat([]byte("x"), maxBlockSize+1))
	for _, lighthouse := range []bool{false, true} {
		serveBlocks(t, lighthouse, block)
		prefix := IpfsPrefix
		if lighthouse {
			prefix = FilecoinPrefix
		}
		data, err := ReadFile(prefix + block.cid.String())
		require.ErrorContains(t, err, "exceeds")
		assert.Nil(t, data)
	}
}

func TestGatewayCredentialsRejected(t *testing.T) {
	withTestConfig(t, map[string]string{config.LighthouseEndpoint: "https://user:password@example.com/ipfs/"})
	data, err := GetLighthouseFile(rawFixture(t, []byte("public")).cid.String())
	require.ErrorContains(t, err, "must not contain credentials")
	assert.Nil(t, data)
}

func TestGatewayRedirects(t *testing.T) {
	t.Setenv("LIGHTHOUSE_API_KEY", "unused-test-api-key")
	block := rawFixture(t, []byte("public content"))
	serveBlocks(t, true, block)
	publicURL := config.GetString(config.LighthouseEndpoint) + block.cid.String() + "?format=raw"
	var target atomic.Value
	target.Store(publicURL)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.Load().(string), http.StatusFound)
	}))
	defer server.Close()
	withTestConfig(t, map[string]string{config.LighthouseEndpoint: server.URL + "/ipfs/"})
	data, err := GetLighthouseFile(block.cid.String())
	require.NoError(t, err)
	assert.Equal(t, "public content", string(data))

	target.Store(strings.Replace(publicURL, "http://", "http://user:password@", 1))
	data, err = GetLighthouseFile(block.cid.String())
	require.ErrorContains(t, err, "redirect must not contain credentials")
	assert.Nil(t, data)
}

func TestIPFSBareEndpoint(t *testing.T) {
	block := rawFixture(t, []byte("bare host:port endpoint"))
	serveBlocks(t, false, block)
	endpoint := strings.TrimPrefix(config.GetString(config.IpfsEndpoint), "http://")
	withTestConfig(t, map[string]string{config.IpfsEndpoint: endpoint})
	data, err := GetIpfsFile(block.cid.String())
	require.NoError(t, err)
	assert.Equal(t, block.data, data)
}

func TestReadFileTimeoutCoversAllBlocks(t *testing.T) {
	child := rawFixture(t, []byte("child"))
	root := unixFSFixture(t, nil, child)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, delay := child.data, 800*time.Millisecond
		if strings.Contains(r.URL.String(), root.cid.String()) {
			data, delay = root.data, 350*time.Millisecond
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			_, _ = w.Write(data)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: server.URL, config.LighthouseEndpoint: server.URL + "/ipfs/", config.IpfsTimeout: "1",
	})
	for _, prefix := range []string{IpfsPrefix, FilecoinPrefix} {
		data, err := ReadFile(prefix + root.cid.String())
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Nil(t, data)
	}
}

func TestIPFSStreamError(t *testing.T) {
	block := rawFixture(t, []byte("incomplete stream"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Trailer", "X-Stream-Error")
		_, _ = w.Write(block.data)
		w.Header().Set("X-Stream-Error", "fixture unavailable")
	}))
	defer server.Close()
	withTestConfig(t, map[string]string{config.IpfsEndpoint: server.URL})
	data, err := GetIpfsFile(block.cid.String())
	require.ErrorContains(t, err, "block stream failed: fixture unavailable")
	assert.Nil(t, data)
}
