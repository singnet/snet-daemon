package utils

import (
	"bytes"
	"encoding/gob"
	"net/url"
	"strings"
)

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

func Deserialize(slice string, value any) (err error) {
	b := bytes.NewBuffer([]byte(slice))
	d := gob.NewDecoder(b)
	return d.Decode(value)
}

func CheckIfHttps(endpoints []string) bool {
	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "https") {
			return true
		}
	}
	return false
}

func IsJWT(token string) bool {
	parts := strings.Split(token, ".")
	// jwt always has 3 parts: header, payload, signature
	if len(parts) != 3 {
		return false
	}
	// check if each part is not empty
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}
	return true
}

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
