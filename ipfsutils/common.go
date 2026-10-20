package ipfsutils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/ipfs/go-cid"
	"github.com/singnet/snet-daemon/v6/config"
)

const (
	IpfsPrefix     = "ipfs://"
	FilecoinPrefix = "filecoin://"
)

func ReadFile(hash string) (rawFile []byte, err error) {
	return ReadFileContext(context.Background(), hash)
}

func ReadFileContext(ctx context.Context, hash string) (rawFile []byte, err error) {
	hash = trimReference(hash)
	if strings.HasPrefix(hash, FilecoinPrefix) {
		return GetLighthouseFileContext(ctx, hash)
	}
	return GetIpfsFileContext(ctx, hash)
}

func trimReference(reference string) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(reference), "\x00"))
}

func parseReference(reference, prefix string) (cid.Cid, error) {
	encoded := strings.TrimPrefix(trimReference(reference), prefix)
	// Some multibase decoders tolerate embedded line breaks. A reference must
	// be a single intact CID, rather than silently repaired input.
	if strings.ContainsRune(encoded, '\x00') || strings.IndexFunc(encoded, unicode.IsSpace) >= 0 {
		return cid.Undef, fmt.Errorf("invalid content reference: embedded whitespace or NUL")
	}
	c, err := cid.Decode(encoded)
	if err != nil {
		return cid.Undef, fmt.Errorf("invalid content reference: %w", err)
	}
	return c, nil
}

func retrievalTimeout(lighthouse bool) time.Duration {
	timeout := time.Duration(config.GetInt(config.IpfsTimeout)) * time.Second
	if lighthouse && timeout <= 0 {
		return 30 * time.Second
	}
	return timeout
}

func retrievalContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func fetchHTTPBlock(ctx context.Context, client *http.Client, method, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.ipld.raw")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("block request returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	// Do not drain an oversized response: the server's bytes are still untrusted.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBlockSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBlockSize {
		return nil, fmt.Errorf("block exceeds the %d byte limit", maxBlockSize)
	}
	if message := resp.Trailer.Get("X-Stream-Error"); message != "" {
		return nil, fmt.Errorf("block stream failed: %s", message)
	}
	return data, nil
}
