package ipfsutils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ipfs/go-cid"

	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

func GetLighthouseFile(cID string) ([]byte, error) {
	return GetLighthouseFileContext(context.Background(), cID)
}

func GetLighthouseFileContext(ctx context.Context, cID string) ([]byte, error) {
	zap.L().Debug("Getting lighthouse file", zap.String("cid", cID))
	root, err := parseReference(cID, FilecoinPrefix)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(config.GetString(config.LighthouseEndpoint))
	if err != nil {
		return nil, fmt.Errorf("invalid gateway endpoint: %w", err)
	}
	if endpoint.User != nil {
		return nil, fmt.Errorf("gateway endpoint must not contain credentials")
	}
	timeout := retrievalTimeout(true)
	ctx, cancel := retrievalContext(ctx, timeout)
	defer cancel()
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.User != nil {
				return fmt.Errorf("gateway redirect must not contain credentials")
			}
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 gateway redirects")
			}
			return nil
		},
	}
	return readVerifiedUnixFSFile(ctx, root, func(ctx context.Context, c cid.Cid) ([]byte, error) {
		u := endpoint.JoinPath(c.String())
		query := u.Query()
		query.Set("format", "raw")
		u.RawQuery = query.Encode()
		return fetchHTTPBlock(ctx, client, http.MethodGet, u.String())
	})
}
