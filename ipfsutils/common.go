package ipfsutils

import (
	"context"
	"regexp"
	"strings"
)

const (
	IpfsPrefix     = "ipfs://"
	FilecoinPrefix = "filecoin://"
)

func ReadFile(hash string) (rawFile []byte, err error) {
	return ReadFileContext(context.Background(), hash)
}

func ReadFileContext(ctx context.Context, hash string) (rawFile []byte, err error) {
	if strings.HasPrefix(hash, FilecoinPrefix) {
		rawFile, err = GetLighthouseFileContext(ctx, formatHash(hash))
	} else {
		rawFile, err = GetIpfsFileContext(ctx, formatHash(hash))
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
