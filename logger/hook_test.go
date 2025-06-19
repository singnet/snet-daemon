package logger

import (
	"errors"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"strconv"
	"testing"
)

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
