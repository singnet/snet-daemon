package blockchain

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testServiceProto = `syntax = "proto3";
package example_service;
message Request { string text = 1; }
message Response { string text = 1; }
service Example { rpc Predict(Request) returns (Response); }
`

func metadataContentCID(t *testing.T, data []byte) string {
	t.Helper()
	digest := sha256.Sum256(data)
	hash, err := mh.Encode(digest[:], mh.SHA2_256)
	require.NoError(t, err)
	return cid.NewCidV1(cid.Raw, hash).String()
}

func setMetadataEndpoint(t *testing.T, key, endpoint string) {
	t.Helper()
	v := config.Vip()
	original := v.Get(key)
	v.Set(key, endpoint)
	t.Cleanup(func() { v.Set(key, original) })
}

func serveMetadataContents(t *testing.T, lighthouse bool, contents ...[]byte) []string {
	t.Helper()
	blocks := make(map[string][]byte)
	var hashes []string
	for _, data := range contents {
		hash := metadataContentCID(t, data)
		blocks[hash] = data
		hashes = append(hashes, hash)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Cookie"))
		var hash string
		if lighthouse {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "raw", r.URL.Query().Get("format"))
			hash = strings.TrimPrefix(r.URL.Path, "/ipfs/")
		} else {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v0/block/get", r.URL.Path)
			hash = r.URL.Query().Get("arg")
		}
		data, ok := blocks[hash]
		if !ok {
			http.Error(w, "fixture unavailable", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.ipld.raw")
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	if lighthouse {
		setMetadataEndpoint(t, config.LighthouseEndpoint, server.URL+"/ipfs/")
	} else {
		setMetadataEndpoint(t, config.IpfsEndpoint, server.URL)
	}
	return hashes
}

func serviceProtoArchive(t *testing.T, compressed bool) []byte {
	t.Helper()
	var archive bytes.Buffer
	w := tar.NewWriter(&archive)
	require.NoError(t, w.WriteHeader(&tar.Header{Name: "service.proto", Mode: 0600, Size: int64(len(testServiceProto))}))
	_, err := w.Write([]byte(testServiceProto))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	if !compressed {
		return archive.Bytes()
	}
	var result bytes.Buffer
	gz := gzip.NewWriter(&result)
	_, err = gz.Write(archive.Bytes())
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return result.Bytes()
}

func serviceMetadataLocalFixture(t *testing.T) string {
	t.Helper()
	hash := serveMetadataContents(t, false, serviceProtoArchive(t, false))[0]
	data, err := os.ReadFile("../resources/testing/service_metadata.json")
	require.NoError(t, err)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(data, &metadata))
	metadata["model_ipfs_hash"] = hash
	data, err = json.Marshal(metadata)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "service_metadata.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	return path
}

func TestInitServiceMetaDataFromJson_ProtoSources(t *testing.T) {
	for _, tc := range []struct {
		field  string
		prefix string
	}{
		{"model_ipfs_hash", ""},
		{"service_api_source", "ipfs://"},
		{"service_api_source", "filecoin://"},
	} {
		for _, compressed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%s/gzip=%t", tc.field, tc.prefix, compressed), func(t *testing.T) {
				archive := serviceProtoArchive(t, compressed)
				hash := serveMetadataContents(t, tc.prefix == "filecoin://", archive)[0]
				var data map[string]any
				require.NoError(t, json.Unmarshal([]byte(testJsonData), &data))
				data[tc.field] = tc.prefix + hash
				payload, err := json.Marshal(data)
				require.NoError(t, err)

				metadata, err := InitServiceMetaDataFromJson(payload)
				require.NoError(t, err)
				assert.Equal(t, map[string]string{"service.proto": testServiceProto}, metadata.ProtoFiles)
			})
		}
	}
}

func TestMetadataRetrieval_VerifiedContents(t *testing.T) {
	for _, lighthouse := range []bool{false, true} {
		prefix := "ipfs://"
		if lighthouse {
			prefix = "filecoin://"
		}
		t.Run(prefix, func(t *testing.T) {
			archive := serviceProtoArchive(t, true)
			var data map[string]any
			require.NoError(t, json.Unmarshal([]byte(testJsonData), &data))
			data["service_api_source"] = prefix + metadataContentCID(t, archive)
			payload, err := json.Marshal(data)
			require.NoError(t, err)
			hashes := serveMetadataContents(t, lighthouse, []byte(testJsonOrgGroupData), payload, archive)

			org, err := GetOrganizationMetaDataFromIPFS(prefix + hashes[0])
			require.NoError(t, err)
			assert.Equal(t, "organization_name", org.OrgName)
			service, err := GetServiceMetaDataFromIPFS(prefix + hashes[1])
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"service.proto": testServiceProto}, service.ProtoFiles)
		})
	}
}

func TestMetadataRetrieval_RejectsForgedContents(t *testing.T) {
	for _, lighthouse := range []bool{false, true} {
		prefix := "ipfs://"
		if lighthouse {
			prefix = "filecoin://"
		}
		t.Run(prefix, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("not JSON or a proto archive"))
			}))
			defer server.Close()
			key, endpoint := config.IpfsEndpoint, server.URL
			if lighthouse {
				key, endpoint = config.LighthouseEndpoint, server.URL+"/ipfs/"
			}
			setMetadataEndpoint(t, key, endpoint)
			hash := metadataContentCID(t, []byte(testJsonOrgGroupData))
			org, err := GetOrganizationMetaDataFromIPFS(prefix + hash)
			require.ErrorContains(t, err, "CID mismatch")
			assert.Nil(t, org)
			service, err := GetServiceMetaDataFromIPFS(prefix + hash)
			require.ErrorContains(t, err, "CID mismatch")
			assert.Nil(t, service)

			var data map[string]any
			require.NoError(t, json.Unmarshal([]byte(testJsonData), &data))
			data["service_api_source"] = prefix + metadataContentCID(t, serviceProtoArchive(t, false))
			payload, err := json.Marshal(data)
			require.NoError(t, err)
			service, err = InitServiceMetaDataFromJson(payload)
			require.ErrorContains(t, err, "CID mismatch")
			assert.Nil(t, service)
		})
	}
}

func TestInitServiceMetaDataFromJson_IPFSError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"Message":"fixture unavailable","Code":1,"Type":"error"}`))
	}))
	t.Cleanup(server.Close)
	setMetadataEndpoint(t, config.IpfsEndpoint, server.URL)
	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(testJsonData), &data))
	data["model_ipfs_hash"] = "Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc"
	payload, err := json.Marshal(data)
	require.NoError(t, err)

	metadata, err := InitServiceMetaDataFromJson(payload)
	require.ErrorContains(t, err, "fixture unavailable")
	assert.Nil(t, metadata)
}
