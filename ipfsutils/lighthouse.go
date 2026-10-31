package ipfsutils

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

func GetLighthouseFile(cID string) ([]byte, error) {
	return GetLighthouseFileContext(context.Background(), cID)
}

func GetLighthouseFileContext(ctx context.Context, cID string) ([]byte, error) {
	zap.L().Debug("Getting lighthouse file", zap.String("cid", cID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.GetString(config.LighthouseEndpoint)+cID, nil)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	file, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return file, nil
}
