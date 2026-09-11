package ipfsutils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	zap.ReplaceGlobals(zap.New(zapcore.NewNopCore()))
}

func withTestConfig(t *testing.T, overrides map[string]string) {
	t.Helper()
	originals := make(map[string]string)
	for k := range overrides {
		originals[k] = config.GetString(k)
	}
	for k, v := range overrides {
		config.Vip().Set(k, v)
	}
	t.Cleanup(func() {
		for k, v := range originals {
			config.Vip().Set(k, v)
		}
	})
}

func startMockServer(handler http.HandlerFunc) *httptest.Server {
	ts := httptest.NewServer(handler)
	return ts
}

func gzipData(data []byte) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(data)
	gw.Close()
	return buf.Bytes()
}

func createTarEntries(entries []struct {
	name    string
	content string
}) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.name,
			Mode: 0600,
			Size: int64(len(e.content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write([]byte(e.content)); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func createTarWithEntry(header *tar.Header, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ========== lighthouse.go ==========

func TestGetLighthouseFile_Success(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("filecoin-content"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{config.LighthouseEndpoint: ts.URL + "/"})

	data, err := GetLighthouseFile("testcid")
	require.NoError(t, err)
	assert.Equal(t, "filecoin-content", string(data))
}

func TestGetLighthouseFile_HTTPError(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{config.LighthouseEndpoint: ts.URL + "/"})

	data, err := GetLighthouseFile("testcid")
	require.NoError(t, err)
	assert.Equal(t, "server error", string(data))
}

func TestGetLighthouseFile_ConnectionRefused(t *testing.T) {
	withTestConfig(t, map[string]string{config.LighthouseEndpoint: "http://127.0.0.1:1/"})

	data, err := GetLighthouseFile("testcid")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetLighthouseFile_ReadBodyError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, bufrw, err := hijacker.Hijack()
		if err != nil {
			return
		}
		fmt.Fprint(bufrw, "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
		fmt.Fprint(bufrw, "6\r\nhello!\r\n")
		bufrw.Flush()
		conn.Close()
	}))
	defer ts.Close()

	withTestConfig(t, map[string]string{config.LighthouseEndpoint: ts.URL + "/"})

	data, err := GetLighthouseFile("testcid")
	assert.Error(t, err)
	assert.Nil(t, data)
}

// ========== common.go (ReadFile) ==========

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

func TestGetIpfsFile_InvalidHash(t *testing.T) {
	withTestConfig(t, map[string]string{config.IpfsEndpoint: "http://127.0.0.1:1"})

	data, err := GetIpfsFile("not-a-valid-cid")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetIpfsFile_SendError(t *testing.T) {
	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: "http://127.0.0.1:1",
		config.IpfsTimeout:  "1",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetIpfsFile_ServerError(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("command not found"))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Nil(t, data)
	assert.Nil(t, err)
}

func TestGetIpfsFile_ServerJSONError(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"Message":"internal error","Code":1}`))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Nil(t, data)
	assert.Nil(t, err)
}

func TestGetIpfsFile_Success(t *testing.T) {
	content := []byte("IPFS file content")
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(content)
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestGetIpfsFile_ReadBodyError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, bufrw, err := hijacker.Hijack()
		if err != nil {
			return
		}
		fmt.Fprint(bufrw, "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
		fmt.Fprint(bufrw, "6\r\nhello!\r\n")
		bufrw.Flush()
		conn.Close()
	}))
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	data, err := GetIpfsFile("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestGetIPFSClient_Success(t *testing.T) {
	ts := startMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Version":"0.1.0"}`))
	})
	defer ts.Close()

	withTestConfig(t, map[string]string{
		config.IpfsEndpoint: ts.URL,
		config.IpfsTimeout:  "10",
	})

	client := GetIPFSClient()
	assert.NotNil(t, client)
}

// ========== compressed.go ==========

func TestReadProtoFilesCompressed_PlainTar(t *testing.T) {
	tarData, err := createTarEntries([]struct{ name, content string }{
		{"service.proto", "syntax = \"proto3\";"},
		{"other.proto", "message Foo {}"},
	})
	require.NoError(t, err)

	protos, err := ReadProtoFilesCompressed(tarData)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"service.proto": "syntax = \"proto3\";",
		"other.proto":   "message Foo {}",
	}, protos)
}

func TestReadProtoFilesCompressed_GzipTar(t *testing.T) {
	tarData, err := createTarEntries([]struct{ name, content string }{
		{"api.proto", "service S {}"},
	})
	require.NoError(t, err)

	protos, err := ReadProtoFilesCompressed(gzipData(tarData))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"api.proto": "service S {}"}, protos)
}

func TestReadProtoFilesCompressed_GzipError(t *testing.T) {
	_, err := ReadProtoFilesCompressed([]byte{0x1F, 0x8B, 0x00, 0x00, 0x00})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decompress gzip")
}

func TestReadProtoFilesCompressed_EmptyTar(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.Close()

	protos, err := ReadProtoFilesCompressed(buf.Bytes())
	require.NoError(t, err)
	assert.Empty(t, protos)
}

func TestReadProtoFilesCompressed_Directory(t *testing.T) {
	tarData, err := createTarWithEntry(&tar.Header{
		Name: "some_dir/", Typeflag: tar.TypeDir, Mode: 0755,
	}, nil)
	require.NoError(t, err)

	protos, err := ReadProtoFilesCompressed(tarData)
	require.NoError(t, err)
	assert.Empty(t, protos)
}

func TestReadProtoFilesCompressed_NonProtoFile(t *testing.T) {
	tarData, err := createTarEntries([]struct{ name, content string }{
		{"readme.txt", "hello"},
		{"code.go", "package main"},
	})
	require.NoError(t, err)

	protos, err := ReadProtoFilesCompressed(tarData)
	require.NoError(t, err)
	assert.Empty(t, protos)
}

func TestReadProtoFilesCompressed_UnknownFileType(t *testing.T) {
	tarData, err := createTarWithEntry(&tar.Header{
		Name: "badfile", Typeflag: tar.TypeSymlink, Linkname: "/tmp", Mode: 0600,
	}, nil)
	require.NoError(t, err)

	_, err = ReadProtoFilesCompressed(tarData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown file type")
}

func TestReadProtoFilesCompressed_MixedEntries(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	tw.WriteHeader(&tar.Header{Name: "data/", Typeflag: tar.TypeDir, Mode: 0755})

	tw.WriteHeader(&tar.Header{Name: "data/readme.txt", Typeflag: tar.TypeReg, Mode: 0600, Size: 4})
	tw.Write([]byte("read"))

	tw.WriteHeader(&tar.Header{Name: "data/api.proto", Typeflag: tar.TypeReg, Mode: 0600, Size: 14})
	tw.Write([]byte("service Api {}"))

	tw.Close()

	protos, err := ReadProtoFilesCompressed(buf.Bytes())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"data/api.proto": "service Api {}"}, protos)
}

func TestReadProtoFilesCompressed_GzipEmptyTar(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.Close()

	protos, err := ReadProtoFilesCompressed(gzipData(buf.Bytes()))
	require.NoError(t, err)
	assert.Empty(t, protos)
}

func TestReadProtoFilesCompressed_TarTruncatedEntry(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "test.proto", Mode: 0600, Size: 100})
	tw.Write([]byte("hello"))
	tw.Close()

	_, err := ReadProtoFilesCompressed(buf.Bytes())
	assert.Error(t, err)
}

func TestReadProtoFilesCompressed_TarNextError(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	tw.WriteHeader(&tar.Header{Name: "file1.proto", Mode: 0600, Size: 5})
	tw.Write([]byte("hello"))

	tw.WriteHeader(&tar.Header{Name: "file2.proto", Mode: 0600, Size: 5})
	tw.Write([]byte("world"))
	tw.Close()

	truncated := buf.Bytes()[:1024+100]

	_, err := ReadProtoFilesCompressed(truncated)
	assert.Error(t, err)
}

func TestReadProtoFilesCompressed_GzipTruncatedEntry(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tw.WriteHeader(&tar.Header{Name: "test.proto", Mode: 0600, Size: 100})
	tw.Write([]byte("hello"))
	tw.Close()

	gzData := gzipData(buf.Bytes())
	truncatedGz := gzData[:len(gzData)-2]

	protos, err := ReadProtoFilesCompressed(truncatedGz)
	assert.Error(t, err)
	assert.Nil(t, protos)
}
