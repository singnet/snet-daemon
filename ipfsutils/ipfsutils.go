package ipfsutils

import (
	"context"
	"errors"

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
	if req == nil {
		zap.L().Error("error executing the cat command in ipfs: req is nil", zap.String("hashFromMetaData", hash))
		return nil, err
	}
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
		zap.L().Error("error executing the cat command in ipfs", zap.String("hashFromMetaData", hash), zap.Error(err))
		return nil, err
	}
	fileContent, err := io.ReadAll(resp.Output)
	if err != nil {
		zap.L().Error("error: in Reading the meta data file", zap.Error(err), zap.String("hashFromMetaData", hash))
		return nil, err
	}

	// Create a cid manually to check cid
	_, c, err := cid.CidFromBytes(append(cID.Bytes(), fileContent...))
	if err != nil {
		zap.L().Error("error generating ipfs hash", zap.String("hashFromMetaData", hash), zap.Error(err))
		return nil, err
	}

	// To test if two cid's are equivalent, be sure to use the 'Equals' method:
	if !c.Equals(cID) {
		zap.L().Error("IPFS hash verification failed. Generated hash doesnt match with expected hash",
			zap.String("expectedHash", hash),
			zap.String("hashFromIPFSContent", c.String()))
		return nil, errors.New("IPFS hash doesnt match with expected hash")
	}

	return fileContent, nil
}

func GetIPFSClient() *rpc.HttpApi {
	httpClient := http.Client{
		Timeout: time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second,
	}
	ifpsClient, err := rpc.NewURLApiWithClient(config.GetString(config.IpfsEndpoint), &httpClient)
	if err != nil {
		zap.L().Fatal("Connection failed to IPFS", zap.String("IPFS", config.GetString(config.IpfsEndpoint)), zap.Error(err))
	}
	return ifpsClient
}
