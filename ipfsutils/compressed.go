package ipfsutils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
)

// ReadProtoFilesCompressed reads proto files from tar or tar.gz archive
func ReadProtoFilesCompressed(compressedFile []byte) (protos map[string]string, err error) {
	var reader io.Reader = bytes.NewReader(compressedFile)

	if isGzipFile(compressedFile) {
		zap.L().Debug("Detected gzip-compressed tar file, decompressing...")
		gzr, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer gzr.Close()
		reader = gzr // Use the decompressed stream
	}

	tarReader := tar.NewReader(reader)
	protos = make(map[string]string)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			zap.L().Error("Failed to read tar entry", zap.Error(err))
			return nil, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			zap.L().Warn("Directory found in archive, daemon don't support dirs", zap.String("name", header.Name))
		case tar.TypeReg:
			zap.L().Debug("File found in archive", zap.String("name", header.Name))
			data, err := io.ReadAll(tarReader)
			if err != nil {
				zap.L().Error("Failed to read file from tar", zap.Error(err))
				return nil, err
			}
			if !strings.HasSuffix(header.Name, ".proto") { // ignoring not proto files
				zap.L().Debug("Detected not .proto file in archive, skipping", zap.String("name", header.Name))
				continue
			}
			protos[header.Name] = string(data)
		default:
			err := fmt.Errorf("unknown file type %c in file %s", header.Typeflag, header.Name)
			zap.L().Error(err.Error())
			return nil, err
		}
	}
	return protos, nil
}

func isGzipFile(data []byte) bool {
	// Gzip files start with the bytes 0x1F 0x8B
	return len(data) > 2 && data[0] == 0x1F && data[1] == 0x8B
}
