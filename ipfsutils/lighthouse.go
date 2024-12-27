package ipfsutils

import (
	"github.com/singnet/snet-daemon/v5/config"
	"io"
	"net/http"
)

func GetLighthouseFile(cID string) ([]byte, error) {
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
