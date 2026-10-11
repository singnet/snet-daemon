package ipfsutils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
