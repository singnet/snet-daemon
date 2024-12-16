package ipfsutils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"io"
)

// ReadFilesCompressed - read all files which have been compressed, there can be more than one file
// We need to start reading the proto files associated with the service.
// proto files are compressed and stored as modelipfsHash
func ReadFilesCompressed(compressedFile []byte) (protos map[string]string, err error) {
	f := bytes.NewReader(compressedFile)
	tarReader := tar.NewReader(f)
	protos = make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			zap.L().Error(err.Error())
			return nil, err
		}
		name := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			zap.L().Debug("Directory name", zap.String("name", name))
		case tar.TypeReg:
			zap.L().Debug("File from archive", zap.String("name", name))
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil && err != io.EOF {
				zap.L().Error(err.Error())
				return nil, err
			}
			protos[name] = string(data)
		default:
			err = fmt.Errorf(fmt.Sprintf("%s : %c %s %s\n",
				"Unknown file Type ",
				header.Typeflag,
				"in file",
				name,
			))
			zap.L().Error(err.Error())
			return nil, err
		}
	}
	return protos, nil
}
