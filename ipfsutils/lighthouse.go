package ipfsutils

import (
	"io"
	"net/http"

	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

func GetLighthouseFile(cID string) ([]byte, error) {
	zap.L().Debug("Getting lighthouse file", zap.String("cid", cID))
	resp, err := http.Get(config.GetString(config.LighthouseEndpoint) + cID)
	if err != nil {
		return nil, err
	}
	file, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return file, nil
}
