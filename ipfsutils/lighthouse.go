package ipfsutils

import (
	"io"
	"net/http"
)

const lighthouseURL = "https://gateway.lighthouse.storage/ipfs/"

func GetLighthouseFile(cID string) ([]byte, error) {
	resp, err := http.Get(lighthouseURL + cID)
	if err != nil {
		return nil, err
	}
	file, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return file, nil
}
