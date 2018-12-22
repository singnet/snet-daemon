package metrics

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestGenXid(t *testing.T) {
	id1 := GenXid()
	id2 := GenXid()
	assert.NotEqual(t, id1, id2)

}

func TestConvertObjectToJSON(t *testing.T) {
	anystrut := struct {
		RequestID string
	}{RequestID: "12345"}
	json, err := ConvertObjectToJSON(anystrut)
	assert.Equal(t, json, "{\"RequestID\":\"12345\"}")
	assert.Equal(t, err, nil)
	json, err = ConvertObjectToJSON(nil)
	assert.Equal(t, "null", json)
}

func TestGetValue(t *testing.T) {
	md := metadata.Pairs("user-agent", "Test user agent", "user-agent", "user-agent", "content-type", "application/grpc")

	assert.Equal(t, "Test user agent", GetValue(md, "user-agent"))

	assert.Equal(t, GetValue(md, ""), "")
	md = metadata.Pairs()
	assert.Equal(t, GetValue(md, ""), "")
}
