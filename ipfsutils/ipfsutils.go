package ipfsutils

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"

	"io"
	"net/http"
	"time"
)

func GetIpfsFile(hash string) (content []byte, err error) {

	zap.L().Debug("getting file from IPFS", zap.String("hash", hash))

	ipfsClient := GetIPFSClient()

	cID, err := cid.Parse(hash)
	if err != nil {
		zap.L().Error("error parsing the ipfs hash", zap.String("hashFromMetaData", hash), zap.Error(err))
		return nil, err
	}

	req := ipfsClient.Request("cat", cID.String())
	resp, err := req.Send(context.Background())
	if err != nil {
		zap.L().Error("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return nil, err
	}
	defer func(resp *rpc.Response) {
		err := resp.Close()
		if err != nil {
			zap.L().Error(err.Error())
		}
	}(resp)

	if resp.Error != nil {
		zap.L().Error("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(resp.Error))
		return nil, resp.Error
	}
	fileContent, err := io.ReadAll(resp.Output)
	if err != nil {
		zap.L().Error("error: in Reading the meta data file", zap.Error(err), zap.String("hashFromMetaData", hash))
		return nil, err
	}

	return fileContent, nil
}

func GetIPFSClient() *rpc.HttpApi {
	httpClient := http.Client{
		Timeout: time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second,
	}
	// NewURLApiWithClient only builds an HTTP client and always returns a nil error.
	// Connection errors are reported when a request is sent.
	ipfsClient, _ := rpc.NewURLApiWithClient(config.GetString(config.IpfsEndpoint), &httpClient)
	return ipfsClient
}
