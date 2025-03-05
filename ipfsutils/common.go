package ipfsutils

import (
	"regexp"
	"strings"
)

const (
	IpfsPrefix     = "ipfs://"
	FilecoinPrefix = "filecoin://"
)

func ReadFile(hash string) (rawFile []byte, err error) {
	if strings.HasPrefix(hash, FilecoinPrefix) {
		rawFile, err = GetLighthouseFile(formatHash(hash))
	} else {
		rawFile, err = GetIpfsFile(formatHash(hash))
	}
	return rawFile, err
}

func formatHash(hash string) string {
	//zap.L().Debug("Before Formatting", zap.String("metadataHash", hash))
	hash = strings.Replace(hash, IpfsPrefix, "", -1)
	hash = strings.Replace(hash, FilecoinPrefix, "", -1)
	hash = removeSpecialCharacters(hash)
	//zap.L().Debug("hash after format", zap.String("metadataHash", hash))
	return hash
}

func removeSpecialCharacters(pString string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9=]")
	return reg.ReplaceAllString(pString, "")
}
