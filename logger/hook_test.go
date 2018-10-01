package logger

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/zbindenren/logrus_mail"
	"testing"
)

func init() {
	RegisterHookType("test-hook", testHookFactoryMethod)
	RegisterHookType("test-hook-error", testHookFactoryMethodReturnError)
}

type testHook struct {
	config *viper.Viper
	log    []*log.Entry
}

func (hook *testHook) Fire(entry *log.Entry) error {
	hook.log = append(hook.log, entry)
	return nil
}

func (hook *testHook) Levels() []log.Level {
	return nil
}

func testHookFactoryMethod(config *viper.Viper) (*Hook, error) {
	var hook = testHook{config: config}
	return &Hook{Delegate: &hook, ExitHandler: func() {}}, nil
}

func testHookFactoryMethodReturnError(config *viper.Viper) (*Hook, error) {
	return nil, errors.New("as expected")
}

func TestHookFireOnError(t *testing.T) {
	const loggerConfigJson = `
	{
		"hooks": [ "some-hook" ],
		"some-hook": {
			"type": "test-hook",
			"levels": ["Error"],
			"config": { 
				"test_config_field": "test config value"
			}
		}
	}`
	var loggerConfig = newConfigFromString(loggerConfigJson, defaultLogConfig)
	var logger = log.New()
	var err = initLogger(logger, loggerConfig)
	assert.Nil(t, err)

	logger.Error("error test")

	assert.Equal(t, 1, len(logger.Hooks[log.ErrorLevel]))
	var hook = logger.Hooks[log.ErrorLevel][0].(*internalHook).delegate.(*testHook)
	assert.Equal(t, "error test", hook.log[0].Message)
	assert.Equal(t, "test config value", hook.config.GetString("test_config_field"))
}

func TestHookNoFireOnInfo(t *testing.T) {
	const loggerConfigJson = `
	{
		"hooks": [ "some-hook" ],
		"some-hook": {
			"type": "test-hook",
			"levels": ["Error"],
			"config": { }
		}
	}`
	var loggerConfig = newConfigFromString(loggerConfigJson, defaultLogConfig)
	var logger = log.New()
	var err = initLogger(logger, loggerConfig)
	assert.Nil(t, err)

	logger.Info("info test")

	assert.Equal(t, 1, len(logger.Hooks[log.ErrorLevel]))
	var hook = logger.Hooks[log.ErrorLevel][0].(*internalHook).delegate.(*testHook)
	assert.Equal(t, 0, len(hook.log))
}

func TestInitLoggerUnknownHook(t *testing.T) {
	const loggerConfigJson = `
	{
		"hooks": [ "some-hook" ]
	}`
	var loggerConfig = newConfigFromString(loggerConfigJson, defaultLogConfig)
	var logger = log.New()

	var err = initLogger(logger, loggerConfig)

	assert.Equal(t, errors.New("Unable to add log hook \"some-hook\", error: No hook definition"), err)
}

func TestInitLoggerUnknownHookType(t *testing.T) {
	const loggerConfigJson = `
	{
		"hooks": [ "some-hook" ],
		"some-hook": {
			"type": "UNKNOWN"
		}
	}`
	var loggerConfig = newConfigFromString(loggerConfigJson, defaultLogConfig)
	var logger = log.New()

	var err = initLogger(logger, loggerConfig)

	assert.Equal(t, errors.New("Unable to add log hook \"some-hook\", error: Unexpected hook type: \"UNKNOWN\""), err)
}

func TestAddHookNoFactoryMethod(t *testing.T) {
	const hookConfigJson = `
	{
		"type": "UNKNOWN",
		"levels": ["Error"],
		"config": { }
	}`
	var hookConfig = newConfigFromString(hookConfigJson, nil)
	var logger = log.New()

	var err = addHookByConfig(logger, hookConfig)

	assert.Equal(t, errors.New("Unexpected hook type: \"UNKNOWN\""), err)
}

func TestAddHookFactoryMethodReturnsError(t *testing.T) {
	const hookConfigJson = `
	{
		"type": "test-hook-error",
		"levels": ["Error"],
		"config": { }
	}`
	var hookConfig = newConfigFromString(hookConfigJson, nil)
	var logger = log.New()

	var err = addHookByConfig(logger, hookConfig)

	assert.Equal(t, errors.New("Cannot create hook instance: as expected"), err)
}

func TestAddHookCannotParseLevels(t *testing.T) {
	const hookConfigJson = `
	{
		"type": "test-hook",
		"levels": ["Error", "UNKNOWN"],
		"config": { }
	}`
	var hookConfig = newConfigFromString(hookConfigJson, nil)
	var logger = log.New()

	var err = addHookByConfig(logger, hookConfig)

	assert.Equal(t, errors.New("Unable parse log level string: \"UNKNOWN\", err: not a valid logrus Level: \"UNKNOWN\""), err)
}

func TestAddHookNoType(t *testing.T) {
	const hookConfigJson = `
	{
		"levels": ["Error", "UNKNOWN"],
		"config": { }
	}`
	var hookConfig = newConfigFromString(hookConfigJson, nil)
	var logger = log.New()

	var err = addHookByConfig(logger, hookConfig)

	assert.Equal(t, errors.New("No hook type in hook config"), err)
}

func TestAddHookNoLevels(t *testing.T) {
	const hookConfigJson = `
	{
		"type": "test-hook",
		"config": { }
	}`
	var hookConfig = newConfigFromString(hookConfigJson, nil)
	var logger = log.New()

	var err = addHookByConfig(logger, hookConfig)

	assert.Equal(t, errors.New("No levels in hook config"), err)
}

func TestAddHookEmptyLevels(t *testing.T) {
	const hookConfigJson = `
	{
		"type": "test-hook",
		"levels": [],
		"config": { }
	}`
	var hookConfig = newConfigFromString(hookConfigJson, nil)
	var logger = log.New()

	var err = addHookByConfig(logger, hookConfig)

	assert.Nil(t, err)
}

func TestNewMailAuthHook(t *testing.T) {
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
	var mailAuthHookConfig = newConfigFromString(mailAuthHookConfigJson, nil)

	var hook *Hook
	hook, err = newMailAuthHook(mailAuthHookConfig)
	assert.Nil(t, err)

	var expectedHook *logrus_mail.MailAuthHook
	expectedHook, err = logrus_mail.NewMailAuthHook(
		"test-application-name",
		"smtp.gmail.com",
		587,
		"from-user@gmail.com",
		"to-user@gmail.com",
		"smtp-username",
		"secret")
	assert.Nil(t, err)
	assert.Equal(t, hook.Delegate, expectedHook)
}

func TestNewMailAuthHookError(t *testing.T) {
	const mailAuthHookConfigJson = `
	{
		"application_name": "test-application-name",
		"host": "smtp.gmail.com",
		"port": 587
	}`
	var mailAuthHookConfig = newConfigFromString(mailAuthHookConfigJson, nil)

	var hook, err = newMailAuthHook(mailAuthHookConfig)

	assert.Equal(t, errors.New("Unable to create instance of mail auth hook: mail: no address"), err)
	assert.Nil(t, hook)
}

func TestNewMailAuthHookNoConfig(t *testing.T) {
	var hook, err = newMailAuthHook(nil)

	assert.Equal(t, errors.New("Unable to create instance of mail auth hook: no config provided"), err)
	assert.Nil(t, hook)
}
