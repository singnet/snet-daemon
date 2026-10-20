package ipfsutils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

func GetIpfsFile(hash string) (content []byte, err error) {
	return GetIpfsFileContext(context.Background(), hash)
}

func GetIpfsFileContext(ctx context.Context, hash string) (content []byte, err error) {

	zap.L().Debug("getting file from IPFS", zap.String("hash", hash))

	cID, err := parseReference(hash, IpfsPrefix)
	if err != nil {
		return nil, err
	}
	address := config.GetString(config.IpfsEndpoint)
	// Match Kubo's support for a bare host:port endpoint.
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	endpoint, err := url.JoinPath(address, "api/v0/block/get")
	if err != nil {
		return nil, fmt.Errorf("invalid IPFS endpoint: %w", err)
	}
	blockURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	timeout := retrievalTimeout(false)
	ctx, cancel := retrievalContext(ctx, timeout)
	defer cancel()
	client := &http.Client{Timeout: timeout}
	return readVerifiedUnixFSFile(ctx, cID, func(ctx context.Context, c cid.Cid) ([]byte, error) {
		u := *blockURL
		query := u.Query()
		query.Set("arg", c.String())
		u.RawQuery = query.Encode()
		return fetchHTTPBlock(ctx, client, http.MethodPost, u.String())
	})
}

func GetIPFSClient() *rpc.HttpApi {
	httpClient := http.Client{
		Timeout: retrievalTimeout(false),
	}
	// NewURLApiWithClient only builds an HTTP client and always returns a nil error.
	// Connection errors are reported when a request is sent.
	ipfsClient, _ := rpc.NewURLApiWithClient(config.GetString(config.IpfsEndpoint), &httpClient)
	return ipfsClient
}
