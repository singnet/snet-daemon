package storage

import (
	"bytes"
	"encoding/gob"
)

// EncodeGob serializes a value for storage using gob.
func EncodeGob(value any) (string, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DecodeGob decodes a value previously serialized with EncodeGob.
func DecodeGob(encoded string, value any) error {
	return gob.NewDecoder(bytes.NewBufferString(encoded)).Decode(value)
}
