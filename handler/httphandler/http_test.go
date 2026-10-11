package httphandler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandlerEchoesRequestBodyWhenPassthroughIsDisabled(t *testing.T) {
	originalPassthrough := config.GetBool(config.PassthroughEnabledKey)
	config.Vip().Set(config.PassthroughEnabledKey, false)
	t.Cleanup(func() { config.Vip().Set(config.PassthroughEnabledKey, originalPassthrough) })

	handler := NewHTTPHandler(nil)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request body"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "request body", response.Body.String())
}

func TestHTTPHandlerProxiesRequestAndEnforcesRateLimit(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.Equal(t, http.MethodPut, request.Method)
		require.Equal(t, "request-id", request.Header.Get("X-Request-ID"))
		writer.Header().Set("X-Upstream", "received")
		_, _ = writer.Write(append([]byte("upstream:"), body...))
	}))
	defer upstream.Close()

	originalPassthrough := config.GetBool(config.PassthroughEnabledKey)
	originalEndpoint := config.GetString(config.ServiceEndpointKey)
	originalRate := config.GetString(config.RateLimitPerMinute)
	originalBurst := config.GetInt(config.BurstSize)
	config.Vip().Set(config.PassthroughEnabledKey, true)
	config.Vip().Set(config.ServiceEndpointKey, upstream.URL)
	config.Vip().Set(config.RateLimitPerMinute, "1")
	config.Vip().Set(config.BurstSize, 1)
	t.Cleanup(func() {
		config.Vip().Set(config.PassthroughEnabledKey, originalPassthrough)
		config.Vip().Set(config.ServiceEndpointKey, originalEndpoint)
		config.Vip().Set(config.RateLimitPerMinute, originalRate)
		config.Vip().Set(config.BurstSize, originalBurst)
	})

	handler := NewHTTPHandler(nil)
	request := httptest.NewRequest(http.MethodPut, "/", strings.NewReader("payload"))
	request.Header.Set("X-Request-ID", "request-id")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "received", response.Header().Get("X-Upstream"))
	require.Equal(t, "upstream:payload", response.Body.String())

	rateLimitedResponse := httptest.NewRecorder()
	handler.ServeHTTP(rateLimitedResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusTooManyRequests, rateLimitedResponse.Code)
}

func TestHTTPHandlerReturnsInternalErrorForInvalidPassthroughEndpoint(t *testing.T) {
	originalPassthrough := config.GetBool(config.PassthroughEnabledKey)
	originalEndpoint := config.GetString(config.ServiceEndpointKey)
	config.Vip().Set(config.PassthroughEnabledKey, true)
	config.Vip().Set(config.ServiceEndpointKey, "://invalid")
	t.Cleanup(func() {
		config.Vip().Set(config.PassthroughEnabledKey, originalPassthrough)
		config.Vip().Set(config.ServiceEndpointKey, originalEndpoint)
	})

	handler := NewHTTPHandler(nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusInternalServerError, response.Code)
}
