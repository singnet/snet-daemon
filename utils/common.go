package utils

import (
	"bytes"
	"encoding/gob"
	"net/url"
	"strings"
)

// Serialize encodes value as gob bytes carried in a string, not as JSON or text.
func Serialize(value any) (slice string, err error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	err = e.Encode(value)
	if err != nil {
		return
	}

	slice = b.String()
	return
}

// Deserialize decodes gob bytes into the supplied destination pointer.
func Deserialize(slice string, value any) (err error) {
	b := bytes.NewBuffer([]byte(slice))
	d := gob.NewDecoder(b)
	return d.Decode(value)
}

// CheckIfHttps reports whether any endpoint contains the case-sensitive substring
// "https". It does not parse URLs or validate that TLS is configured.
func CheckIfHttps(endpoints []string) bool {
	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "https") {
			return true
		}
	}
	return false
}

// IsJWT recognizes three nonempty dot-separated segments for RPC auth selection.
// It does not decode claims or verify a signature and is not an authentication check.
func IsJWT(token string) bool {
	parts := strings.Split(token, ".")
	// Match the compact signed-token shape: header.payload.signature.
	if len(parts) != 3 {
		return false
	}
	// All three segments must be nonempty for this heuristic.
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}
	return true
}

// IsURLValid accepts HTTP(S) request URIs with a nonempty host. It does not resolve
// the host, connect to the endpoint, or verify a server certificate.
func IsURLValid(endpoint string) bool {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
