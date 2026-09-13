package metrics

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// These helpers replace process-wide state; tests using them must stay sequential.
func isolatedMetricsConfig(t *testing.T) *viper.Viper {
	t.Helper()
	oldConfig, oldToken, oldGroup := config.Vip(), daemonAuthorizationToken, daemonGroupId
	t.Cleanup(func() {
		config.SetVip(oldConfig)
		daemonAuthorizationToken, daemonGroupId = oldToken, oldGroup
	})
	v := viper.New()
	v.Set(config.PvtKeyForMetering, strings.Repeat("0", 63)+"1")
	v.Set(config.OrganizationId, "test-org")
	v.Set(config.ServiceId, "test-service")
	config.SetVip(v)
	daemonAuthorizationToken, daemonGroupId = "test-token", "test-group"
	return v
}

type metricsRoundTripper func(*http.Request) (*http.Response, error)

func (f metricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubMetricsHTTP(t *testing.T, handler metricsRoundTripper) {
	t.Helper()
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	http.DefaultTransport = handler
}

type trackedMetricsBody struct {
	io.Reader
	closed     bool
	closeCalls int
	closeErr   error
}

func (b *trackedMetricsBody) Close() error {
	b.closed = true
	b.closeCalls++
	return b.closeErr
}

type failedMetricsReader struct{}

func (failedMetricsReader) Read([]byte) (int, error) { return 0, errors.New("body read failed") }

func TestConvertStructToJSON(t *testing.T) {
	payload := struct {
		Count int    `json:"count"`
		Name  string `json:"name"`
	}{Count: 3, Name: "example"}
	data, err := ConvertStructToJSON(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"count":3,"name":"example"}`, string(data))
	data, err = ConvertStructToJSON(nil)
	require.ErrorContains(t, err, "empty payload")
	require.Nil(t, data)
	data, err = ConvertStructToJSON(make(chan int))
	var unsupported *json.UnsupportedTypeError
	require.ErrorAs(t, err, &unsupported)
	require.Nil(t, data)
}

func TestPublishSignedRequest(t *testing.T) {
	for _, statusCode := range []int{http.StatusOK, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			isolatedMetricsConfig(t)
			stats := &CommonStats{UserName: "alice", GroupID: "group-a", OrganizationID: "stale-org", ServiceID: "stale-service"}
			body := &trackedMetricsBody{Reader: strings.NewReader("response")}
			calls := 0
			stubMetricsHTTP(t, func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "https://metrics.example.test/usage", req.URL.String())
				data, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.NoError(t, req.Body.Close())
				require.JSONEq(t, `{"count":3}`, string(data))
				for name, value := range map[string]string{
					"Content-Type": "application/json", "X-Daemonid": GetDaemonID(), "X-Token": "test-token",
					"X-username": "alice", "X-Organizationid": "test-org", "X-Serviceid": "test-service",
					"X-Groupid": "group-a", "X-Currentblocknumber": "123",
				} {
					require.Equal(t, value, req.Header.Get(name), name)
				}
				signature, err := base64.StdEncoding.DecodeString(req.Header.Get("X-Signature"))
				require.NoError(t, err)
				require.Len(t, signature, 65)
				// Reconstruct the protocol message independently of the signing function.
				message := append([]byte("_usagealicetest-orgtest-servicegroup-a"), big.NewInt(123).FillBytes(make([]byte, 32))...)
				signer, err := utils.GetSignerAddressFromMessage(message, signature)
				require.NoError(t, err)
				require.Equal(t, common.HexToAddress("0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"), *signer)
				return &http.Response{StatusCode: statusCode, Body: body}, nil
			})
			ok := Publish(map[string]int{"count": 3}, "https://metrics.example.test/usage", stats, big.NewInt(123))
			require.Equal(t, statusCode == http.StatusOK, ok)
			require.Equal(t, 1, calls, "a failed response must not cause an unbounded retry")
			require.Equal(t, "test-org", stats.OrganizationID)
			require.Equal(t, "test-service", stats.ServiceID)
			require.True(t, body.closed, "response body must be closed on success and on HTTP error")
			require.Equal(t, 1, body.closeCalls)
		})
	}
}

func TestCheckSuccessfulResponseClosesBody(t *testing.T) {
	for _, statusCode := range []int{http.StatusOK, http.StatusNoContent, http.StatusForbidden, http.StatusInternalServerError} {
		for _, closeFails := range []bool{false, true} {
			name := http.StatusText(statusCode)
			if closeFails {
				name += "/close error"
			}
			t.Run(name, func(t *testing.T) {
				body := &trackedMetricsBody{Reader: strings.NewReader("response")}
				if closeFails {
					body.closeErr = errors.New("close failed")
				}
				ok, retry := checkForSuccessfulResponse(&http.Response{StatusCode: statusCode, Body: body})
				require.Equal(t, statusCode == http.StatusOK, ok, "closing the body must not change HTTP status handling")
				require.False(t, retry)
				require.Equal(t, 1, body.closeCalls, "every response body must be closed exactly once")
			})
		}
	}
}

func TestSendRequestRejectsInvalidURL(t *testing.T) {
	stubMetricsHTTP(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid URL must not reach the transport")
		return nil, nil
	})
	response, err := sendRequest([]byte(`{}`), "http://%zz", nil, nil)
	require.Error(t, err)
	require.Nil(t, response)
}

func TestSignMessageForMeteringRejectsInvalidKey(t *testing.T) {
	isolatedMetricsConfig(t).Set(config.PvtKeyForMetering, "invalid")
	req, err := http.NewRequest(http.MethodPost, "https://metrics.example.test", nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	SignMessageForMetering(req, &CommonStats{UserName: "alice"}, big.NewInt(123))
	require.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, req.Header,
		"invalid key must not produce authentication headers")
}

func TestGetTokenFromResponse(t *testing.T) {
	token, ok := getTokenFromResponse(nil)
	require.False(t, ok)
	require.Empty(t, token)
	for _, tc := range []struct {
		name   string
		status int
		reader io.Reader
		want   string
	}{
		{"valid token", http.StatusOK, strings.NewReader(`{"data":{"token":"generated-token"}}`), "generated-token"},
		{"forbidden", http.StatusForbidden, strings.NewReader(`{"data":{"token":"must-not-be-used"}}`), ""},
		{"read error", http.StatusOK, failedMetricsReader{}, ""},
		{"invalid JSON", http.StatusOK, strings.NewReader("{broken"), ""},
		{"invalid token type", http.StatusOK, strings.NewReader(`{"data":{"token":123}}`), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &trackedMetricsBody{Reader: tc.reader}
			token, ok := getTokenFromResponse(&http.Response{StatusCode: tc.status, Body: body})
			require.Equal(t, tc.want != "", ok)
			require.Equal(t, tc.want, token)
			require.True(t, body.closed, "the response body must be closed on success and on error")
		})
	}
}

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

func TestPublish(t *testing.T) {
	isolatedMetricsConfig(t)
	calls := 0
	stubMetricsHTTP(t, func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("transport unavailable")
	})
	status := Publish(nil, "", nil, big.NewInt(0))
	assert.Equal(t, status, false)
	status = Publish(nil, "http://localhost:8080", nil, big.NewInt(0))
	assert.Equal(t, status, false)

	status = Publish(struct {
		Title string `json:"title"`
	}{
		Title: "abcd",
	}, "https://metrics.example.test", &CommonStats{}, big.NewInt(0))
	assert.Equal(t, status, false)
	require.Equal(t, 1, calls, "invalid payloads must not send a request")
}

func TestCheckSuccessfulResponse(t *testing.T) {
	status, _ := checkForSuccessfulResponse(nil)
	assert.Equal(t, status, false)
	status, _ = checkForSuccessfulResponse(&http.Response{StatusCode: http.StatusForbidden})
	assert.Equal(t, status, false)
}

//func TestGetSize(t *testing.T) {
//	strt1 := struct {
//		title string
//	}{
//		title: "abcd",
//	}
//	assert.Equal(t, strconv.FormatUint(GetSize(strt1), 10), "20")
//
//	strt2 := struct {
//		title string
//	}{
//		title: "abcdeefffffffffffffffff",
//	}
//	assert.Equal(t, strconv.FormatUint(GetSize(strt2), 10), "39")
//}

func TestGetEpochTime(t *testing.T) {
	before := time.Now().Unix()
	currentEpoch := getEpochTime()
	after := time.Now().Unix()
	require.GreaterOrEqual(t, currentEpoch, before)
	require.LessOrEqual(t, currentEpoch, after)
}

func Test_getPrivateKeyForMetering(t *testing.T) {
	isolatedMetricsConfig(t)
	key, err := getPrivateKeyForMetering()
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf", crypto.PubkeyToAddress(key.PublicKey).Hex())
	signature := signForMeteringValidation(key, big.NewInt(123), MeteringPrefix, &CommonStats{UserName: "test-user"})
	message := append([]byte("_usagetest-user"), big.NewInt(123).FillBytes(make([]byte, 32))...)
	signer, err := utils.GetSignerAddressFromMessage(message, signature)
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), *signer)

	config.Vip().Set(config.PvtKeyForMetering, "invalid")
	key, err = getPrivateKeyForMetering()
	require.Error(t, err)
	require.Nil(t, key)
	config.Vip().Set(config.PvtKeyForMetering, "")
	key, err = getPrivateKeyForMetering()
	require.NoError(t, err)
	require.Nil(t, key)
}
