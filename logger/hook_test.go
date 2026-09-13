package logger

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type recordingHook struct{ entries []zapcore.Entry }

func (h *recordingHook) call(entry zapcore.Entry) error {
	h.entries = append(h.entries, entry)
	return nil
}

func TestInitializeFiltersHookLevels(t *testing.T) {
	setupConfig(t)
	recorder := &recordingHook{}
	const hookType = "recording-coverage-test"
	RegisterHookType(hookType, func(*viper.Viper) (hook, error) { return recorder, nil })
	t.Cleanup(func() { delete(hookFactoryMethodsByType, hookType) })
	vip.Set(LogHooksKey, []string{"capture"})
	vip.Set("log.capture", map[string]any{"type": hookType, "levels": []string{"warn", "error"}})
	Initialize()
	zap.L().Info("not selected")
	zap.L().Warn("warning message")
	zap.L().Error("error message")
	require.Len(t, recorder.entries, 2)
	require.Equal(t, zap.WarnLevel, recorder.entries[0].Level)
	require.Equal(t, "warning message", recorder.entries[0].Message)
	require.Equal(t, zap.ErrorLevel, recorder.entries[1].Level)
	require.Equal(t, "error message", recorder.entries[1].Message)
}

func TestHookMissingConfiguration(t *testing.T) {
	option, err := initHookByConfig(nil)
	require.ErrorContains(t, err, "no hook definition")
	require.Nil(t, option)
	h, err := newTelegramBotHook(nil)
	require.Error(t, err)
	require.Nil(t, h)
}

func TestMailHookDefaultsUsernameToSender(t *testing.T) {
	v := config.NewJsonConfigFromString(`{"from":"sender@example.test","to":"recipient@example.test",
		"host":"localhost","port":25,"password":"test-password"}`)
	h, err := newMailAuthHook(v)
	require.NoError(t, err)
	mail := h.(*emailHook)
	require.Equal(t, "sender@example.test", mail.Username)
	require.Equal(t, "recipient@example.test", mail.To)
	require.Equal(t, 25, mail.Port)
}

type hookTransport func(*http.Request) (*http.Response, error)

func (f hookTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type hookResponseBody struct {
	io.Reader
	closed bool
	err    error
}

func (b *hookResponseBody) Close() error { b.closed = true; return b.err }

func TestTelegramHookRequestAndErrors(t *testing.T) {
	sentinel := errors.New("test transport failure")
	closeErr := errors.New("test close failure")
	for _, tc := range []struct {
		name                  string
		status                int
		transportErr, bodyErr error
		wantError             string
	}{
		{"success", http.StatusOK, nil, nil, ""},
		{"HTTP error", http.StatusForbidden, nil, nil, "response status code is not 200"},
		{"transport error", 0, sentinel, nil, "failed to send HTTP request"},
		{"close error", http.StatusOK, nil, closeErr, "test close failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupConfig(t)
			vip.Set(config.OrganizationId, "test-org")
			vip.Set(config.ServiceId, "test-service")
			oldClient := http.DefaultClient
			t.Cleanup(func() { http.DefaultClient = oldClient })
			body := &hookResponseBody{Reader: strings.NewReader(`{"ok":true}`), err: tc.bodyErr}
			entry := zapcore.Entry{
				Time:  time.Date(2024, 1, 2, 3, 4, 5, 0, time.FixedZone("test", 3600)),
				Level: zap.ErrorLevel, Message: "failure message", Stack: "test stack",
				Caller: zapcore.EntryCaller{Defined: true, File: "/src/daemon.go", Line: 42},
			}
			calls := 0
			var requestContext context.Context
			http.DefaultClient = &http.Client{Transport: hookTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				requestContext = req.Context()
				_, hasDeadline := requestContext.Deadline()
				require.True(t, hasDeadline, "hook requests must have a deadline")
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "https://api.telegram.org/bottest-key/sendMessage", req.URL.String())
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				var payload struct {
					ChatID int64  `json:"chat_id"`
					Text   string `json:"text"`
					Silent bool   `json:"disable_notification"`
				}
				require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
				require.NoError(t, req.Body.Close())
				require.Equal(t, int64(-123), payload.ChatID)
				require.True(t, payload.Silent)
				for _, field := range []string{"OrgID: test-org", "ServiceID: test-service", "Log Level: error",
					"UTC Time: 2024-01-02 02:04:05 +0000 UTC", "FullPath: /src/daemon.go:42",
					"Message: failure message", "Stack: test stack"} {
					require.Contains(t, payload.Text, field)
				}
				if tc.transportErr != nil {
					return nil, tc.transportErr
				}
				return &http.Response{StatusCode: tc.status, Body: body}, nil
			})}
			h := telegramBotHook{ChatID: -123, APIKey: "test-key", DisableNotification: true}
			err := h.call(entry)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
			}
			if tc.transportErr != nil {
				require.ErrorIs(t, err, tc.transportErr)
			} else {
				require.True(t, body.closed)
			}
			require.Equal(t, 1, calls)
			require.ErrorIs(t, requestContext.Err(), context.Canceled)
		})
	}
}

func TestTelegramHookRejectsInvalidURL(t *testing.T) {
	h := telegramBotHook{ChatID: 123, APIKey: "invalid\nkey"}
	require.Error(t, h.call(zapcore.Entry{}))
}

func TestEmailHookRejectsAddressWithNewline(t *testing.T) {
	setupConfig(t)
	h := emailHook{Host: "localhost", Port: 25, Username: "sender", Password: "test-password",
		From: "sender@example.test\r\nInjected: header", To: "recipient@example.test"}
	// net/smtp validates addresses before dialing, so this never sends mail.
	require.ErrorContains(t, h.call(zapcore.Entry{Message: "test message"}), "CR or LF")
}

func init() {
	RegisterHookType("test-hook", newTestHook) // for tests only
}

type testHook struct {
	config *viper.Viper
}

func (t testHook) call(entry zapcore.Entry) error {
	return nil
}

func newTestHook(config *viper.Viper) (hook, error) {
	if config == nil {
		return nil, errors.New("unable to create instance of test hook: no config provided")
	}
	return &testHook{
		config: config,
	}, nil
}

func TestHooksInitError(t *testing.T) {

	tests := []struct {
		hookConf string
		wantErr  error
	}{
		{`{
		"telegram_api_key": "7358436602:xxx",
      	"telegram_chat_id": -103263970,
      	"disable_notification": true,
      	"type": "telegram_bot",
      	"levels": ["warn","error","panic"]}`, nil},
		{`{
		"telegram_api_key": "7358436602:xxx",
      	"telegram_chat_id": 0,
      	"disable_notification": true,
      	"type": "telegram_bot",
      	"levels": [ "warn","error","panic"
	      ]}`, InvalidTelegramBotHookConf},
		{`{"type": "telegram_bot", "levels": ["error"]}`, InvalidTelegramBotHookConf},
		{`{"type": "email", "levels": ["error"]}`, InvalidMailHookConf},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, gotErr := initHookByConfig(config.NewJsonConfigFromString(tt.hookConf))
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("initHookByConfig() = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestInitLoggerUnknownHookType(t *testing.T) {
	const loggerConfigJson = `
	{
		"type": "UNKNOWN",
		"levels":["error"]
	}`
	var conf = config.NewJsonConfigFromString(loggerConfigJson)
	_, err := initHookByConfig(conf)
	assert.Equal(t, errors.New("unexpected hook type: \"UNKNOWN\""), err)
}

func TestAddHookCannotParseLevels(t *testing.T) {
	const hookConfigJson = `
	{
		"type": "telegram_bot",
		"levels": ["error", "UNKNOWN"],
		"telegram_api_key":"123",
		"telegram_chat_id":1
	}`
	var hookConfig = config.NewJsonConfigFromString(hookConfigJson)
	_, err := initHookByConfig(hookConfig)
	assert.Equal(t, errors.New("unable parse log level string: \"UNKNOWN\", err: wrong string for level: UNKNOWN. Available options: debug, info, warn, error, panic"), err)
}

func TestAddHookNoType(t *testing.T) {
	const hookConfigJson = `
		{
			"levels": ["error", "warn"],
			"port": 587,
			"config": { }
		}`
	var hookConfig = config.NewJsonConfigFromString(hookConfigJson)
	_, err := initHookByConfig(hookConfig)
	assert.Equal(t, errors.New("no hook type in hook config"), err)
}

func TestAddHookNoLevels(t *testing.T) {
	const hookConfigJson = `
		{
			"type": "test-hook",
			"config": { }
		}`
	var hookConfig = config.NewJsonConfigFromString(hookConfigJson)
	_, err := initHookByConfig(hookConfig)
	assert.Equal(t, NoLevelsSpecifiedError, err)
}

func TestAddHookEmptyLevels(t *testing.T) {
	const hookConfigJson = `
		{
			"type": "test-hook",
			"levels": [],
			"config": { }
		}`
	var hookConfig = config.NewJsonConfigFromString(hookConfigJson)
	_, err := initHookByConfig(hookConfig)
	assert.Equal(t, NoLevelsSpecifiedError, err)
}

func TestNewMailHook(t *testing.T) {
	var err error
	const mailAuthHookConfigJson = `
		{
			"application_name": "test-application-name",
			"host": "smtp.gmail.com",
			"port": 587,
			"from": "from-user@gmail.com",
			"to": "to-user@gmail.com",
			"username": "smtp-username",
			"password": "secret"
		}`
	var mailAuthHookConfig = config.NewJsonConfigFromString(mailAuthHookConfigJson)
	hook, err := newMailAuthHook(mailAuthHookConfig)
	assert.Nil(t, err)
	assert.NotNil(t, hook)
}

func TestNewMailHookInvalidPort(t *testing.T) {
	const mailAuthHookConfigJson = `
		{
			"application_name": "test-application-name",
			"host": "smtp.gmail.com",
			"port": "port",
			"from": "from-user@gmail.com",
			"to": "to-user@gmail.com",
			"username": "smtp-username",
			"password": "secret"
		}`
	var mailAuthHookConfig = config.NewJsonConfigFromString(mailAuthHookConfigJson)
	_, err := newMailAuthHook(mailAuthHookConfig)
	assert.NotNil(t, err)
}

func TestNewMailAuthHookError(t *testing.T) {
	const mailAuthHookConfigJson = `
	{
		"application_name": "test-application-name",
		"host": "smtp.gmail.com",
		"port": 587
	}`
	var mailAuthHookConfig = config.NewJsonConfigFromString(mailAuthHookConfigJson)
	var hook, err = newMailAuthHook(mailAuthHookConfig)
	assert.Equal(t, errors.New("unable to create instance of mail auth hook: invalid configuration"), err)
	assert.Nil(t, hook)
}

func TestNewMailAuthHookNoConfig(t *testing.T) {
	var hook, err = newMailAuthHook(nil)
	assert.Equal(t, errors.New("unable to create instance of mail auth hook: no config provided"), err)
	assert.Nil(t, hook)
}
