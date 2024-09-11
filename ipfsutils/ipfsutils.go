package ipfsutils

import (
	"archive/tar"
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/singnet/snet-daemon/config"
	"go.uber.org/zap"

	"io"
	"net/http"
	"strings"
	"time"
)

// ReadFilesCompressed - read all files which have been compressed, there can be more than one file
// We need to start reading the proto files associated with the service.
// proto files are compressed and stored as modelipfsHash
func ReadFilesCompressed(compressedFile string) (protofiles []string, err error) {
	f := strings.NewReader(compressedFile)
	tarReader := tar.NewReader(f)
	protofiles = make([]string, 0)
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
			zap.L().Debug("Directory name", zap.Any("name", name))
		case tar.TypeReg:
			zap.L().Debug("File name", zap.Any("name", name))
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil && err != io.EOF {
				zap.L().Error(err.Error())
				return nil, err
			}
			protofiles = append(protofiles, string(data))
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
	return protofiles, nil
}

func GetIpfsFile(hash string) (content string) {

	zap.L().Debug("Hash Used to retrieve from IPFS", zap.String("hash", hash))

	ipfsClient := GetIPFSClient()

	cID, err := cid.Parse(hash)
	if err != nil {
		zap.L().Fatal("error parsing the ipfs hash", zap.String("hashFromMetaData", hash), zap.Error(err))
	}

	req := ipfsClient.Request("cat", cID.String())
	if err != nil {
		zap.L().Fatal("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}
	resp, err := req.Send(context.Background())
	if err != nil {
		zap.L().Fatal("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}
	defer resp.Close()

	if resp.Error != nil {
		zap.L().Fatal("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}
	fileContent, err := io.ReadAll(resp.Output)
	if err != nil {
		zap.L().Fatal("error: in Reading the meta data file", zap.Error(err), zap.String("hashFromMetaData", hash))
		return
	}

	// log.WithField("hash", hash).WithField("blob", string(fileContent)).Debug("Blob of IPFS file with hash")

	// Create a cid manually to check cid
	_, c, err := cid.CidFromBytes(append(cID.Bytes(), fileContent...))
	if err != nil {
		zap.L().Fatal("error generating ipfs hash", zap.String("hashFromMetaData", hash), zap.Error(err))
		return
	}

	// To test if two cid's are equivalent, be sure to use the 'Equals' method:
	if !c.Equals(cID) {
		zap.L().Fatal("IPFS hash verification failed. Generated hash doesnt match with expected hash",
			zap.String("expectedHash", hash),
			zap.String("hashFromIPFSContent", c.String()))
	}

	return string(fileContent)
}

func GetIPFSClient() *rpc.HttpApi {
	httpClient := http.Client{
		Timeout: time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second,
	}
	ifpsClient, err := rpc.NewURLApiWithClient(config.GetString(config.IpfsEndPoint), &httpClient)
	if err != nil {
		zap.L().Panic("Connection failed to IPFS", zap.String("IPFS", config.GetString(config.IpfsEndPoint)), zap.Error(err))
	}
	return ifpsClient
}
