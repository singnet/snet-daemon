package blockchain

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func setMetadataIPFSEndpoint(t *testing.T, endpoint string) {
	t.Helper()
	v := config.Vip()
	original := v.Get(config.IpfsEndpoint)
	v.Set(config.IpfsEndpoint, endpoint)
	t.Cleanup(func() { v.Set(config.IpfsEndpoint, original) })
}

func serveServiceProto(t *testing.T, hash string) {
	t.Helper()
	var archive bytes.Buffer
	w := tar.NewWriter(&archive)
	require.NoError(t, w.WriteHeader(&tar.Header{Name: "service.proto", Mode: 0600, Size: int64(len(testServiceProto))}))
	_, err := w.Write([]byte(testServiceProto))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v0/cat", r.URL.Path)
		assert.Equal(t, hash, r.URL.Query().Get("arg"))
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(archive.Bytes())
	}))
	t.Cleanup(server.Close)
	setMetadataIPFSEndpoint(t, server.URL)
}

func TestInitServiceMetaDataFromJson_ProtoSources(t *testing.T) {
	const hash = "Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc"
	for _, tc := range []struct {
		field string
		value string
	}{{"model_ipfs_hash", hash}, {"service_api_source", "ipfs://" + hash}} {
		t.Run(tc.field, func(t *testing.T) {
			serveServiceProto(t, hash)
			var data map[string]any
			require.NoError(t, json.Unmarshal([]byte(testJsonData), &data))
			data[tc.field] = tc.value
			payload, err := json.Marshal(data)
			require.NoError(t, err)

			metadata, err := InitServiceMetaDataFromJson(payload)
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"service.proto": testServiceProto}, metadata.ProtoFiles)
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
	setMetadataIPFSEndpoint(t, server.URL)
	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(testJsonData), &data))
	data["model_ipfs_hash"] = "Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc"
	payload, err := json.Marshal(data)
	require.NoError(t, err)

	metadata, err := InitServiceMetaDataFromJson(payload)
	require.ErrorContains(t, err, "fixture unavailable")
	assert.Nil(t, metadata)
}
