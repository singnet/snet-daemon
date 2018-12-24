package metrics

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strconv"
	"testing"
)

func TestGenXid(t *testing.T) {
	id1 := GenXid()
	id2 := GenXid()
	assert.NotEqual(t, id1, id2)

}
func TestGetValue(t *testing.T) {
	md := metadata.Pairs("user-agent", "Test user agent", "user-agent", "user-agent", "content-type", "application/grpc")
	assert.Equal(t, "Test user agent", GetValue(md, "user-agent"))
	assert.Equal(t, GetValue(md, ""), "")
	md = metadata.Pairs()
	assert.Equal(t, GetValue(md, ""), "")
}

func TestIsValidUrl(t *testing.T) {
	valid := isValidUrl("")
	assert.Equal(t, valid, false)
	valid = isValidUrl("http://test:8080")
	assert.Equal(t, valid, true)
}
func TestPublish(t *testing.T) {
	status := Publish(nil, "")
	assert.Equal(t, status, false)
	status = Publish(nil, "http://localhost:8080")
	assert.Equal(t, status, false)

	status = Publish(struct {
		title string
	}{
		title: "abcd",
	}, "http://localhost:8080")
	assert.Equal(t, status, false)
}

func TestCheckSuccessfulResponse(t *testing.T) {
	status := checkForSuccessfulResponse(nil)
	assert.Equal(t, status, false)
	status = checkForSuccessfulResponse(&http.Response{StatusCode: http.StatusForbidden})
	assert.Equal(t, status, false)

}

func TestGetSize(t *testing.T) {
	strt1 := struct {
		title string
	}{
		title: "abcd",
	}
	assert.Equal(t, strconv.FormatUint(GetSize(strt1), 10), "20")

	strt2 := struct {
		title string
	}{
		title: "abcdeefffffffffffffffff",
	}
	assert.Equal(t, strconv.FormatUint(GetSize(strt2), 10), "39")
}
