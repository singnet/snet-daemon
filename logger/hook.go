package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"net/http"
	"net/smtp"
	"slices"
	"time"
)

var InvalidMailHookConf = errors.New("unable to create instance of mail auth hook: invalid configuration")
var InvalidTelegramBotHookConf = errors.New("unable to create instance of telegram bot hook: invalid configuration")
var NoLevelsSpecifiedError = errors.New("no levels in hook config")

const (
	LogHookTypeKey   = "type"
	LogHookLevelsKey = "levels"
	LogHooksKey      = "log.hooks"

	LogHookMailApplicationNameKey   = "application_name"
	LogHookMailHostKey              = "host"
	LogHookMailPortKey              = "port"
	LogHookMailFromKey              = "from"
	LogHookMailToKey                = "to"
	LogHookMailUsernameKey          = "username"
	LogHookMailPasswordKey          = "password"
	LogHookTelegramAPIKey           = "telegram_api_key"
	LogHookTelegramChatID           = "telegram_chat_id"
	LogHookTelegramDisNotifications = "telegram_disable_notifications"
)

// RegisterHookType registers new hook type in the system.
func RegisterHookType(hookType string, f func(config *viper.Viper) (hook, error)) {
	hookFactoryMethodsByType[hookType] = f
}

var hookFactoryMethodsByType = map[string]func(config *viper.Viper) (hook, error){}

func init() {
	RegisterHookType("email", newMailAuthHook)
	RegisterHookType("telegram_bot", newTelegramBotHook)
}

type hook interface {
	call(entry zapcore.Entry) error
}

func initHookByConfig(conf *viper.Viper) (hook zap.Option, err error) {

	if conf == nil {
		return nil, errors.New("no hook definition")
	}

	var hookType = conf.GetString(LogHookTypeKey)
	if hookType == "" {
		return nil, errors.New("no hook type in hook config")
	}

	hookInit, ok := hookFactoryMethodsByType[hookType]
	if !ok {
		return nil, fmt.Errorf("unexpected hook type: \"%v\"", hookType)
	}

	internalHook, err := hookInit(conf)
	if err != nil {
		return nil, err
	}

	if conf.Get(LogHookLevelsKey) == nil {
		return nil, NoLevelsSpecifiedError
	}

	var levels []zapcore.Level
	for _, levelString := range conf.GetStringSlice(LogHookLevelsKey) {
		var level zapcore.Level
		level, err = getLoggerLevel(levelString)
		if err != nil {
			return nil, fmt.Errorf("unable parse log level string: \"%v\", err: %v", levelString, err)
		}
		levels = append(levels, level)
	}

	if len(levels) == 0 {
		return nil, NoLevelsSpecifiedError
	}

	return zap.Hooks(func(entry zapcore.Entry) error {
		if slices.Contains(levels, entry.Level) {
			return internalHook.call(entry)
		}
		return nil
	}), nil
}

type telegramBotHook struct {
	ChatID              int64
	APIKey              string
	DisableNotification bool
}

type emailHook struct {
	Username        string
	From            string
	Password        string
	To              string
	Host            string
	Port            int
	applicationName string
}

const sendTGMessageURLTemplate = "https://api.telegram.org/bot%s/sendMessage"

// message is JSON payload representation sent to Telegram API.
type telegramMessage struct {
	ChatID              int64  `json:"chat_id"`
	Text                string `json:"text"`
	DisableNotification bool   `json:"disable_notification"`
}

func newMailAuthHook(config *viper.Viper) (hook, error) {
	if config == nil {
		return nil, errors.New("unable to create instance of mail auth hook: no config provided")
	}
	hook := &emailHook{
		applicationName: config.GetString(LogHookMailApplicationNameKey),
		From:            config.GetString(LogHookMailFromKey),
		Host:            config.GetString(LogHookMailHostKey),
		Port:            config.GetInt(LogHookMailPortKey),
		Password:        config.GetString(LogHookMailPasswordKey),
		To:              config.GetString(LogHookMailToKey),
		Username:        config.GetString(LogHookMailUsernameKey),
	}
	if hook.Username == "" {
		hook.Username = hook.From
	}
	if hook.Password == "" || hook.From == "" || hook.Host == "" || hook.To == "" || hook.Port == 0 {
		return nil, InvalidMailHookConf
	}
	return hook, nil
}

func newTelegramBotHook(config *viper.Viper) (hook, error) {
	if config == nil {
		return nil, errors.New("unable to create instance of mail auth hook: no config provided")
	}
	hook := telegramBotHook{
		ChatID:              config.GetInt64(LogHookTelegramChatID),
		APIKey:              config.GetString(LogHookTelegramAPIKey),
		DisableNotification: config.GetBool(LogHookTelegramDisNotifications),
	}
	if hook.ChatID == 0 || hook.APIKey == "" {
		return nil, InvalidTelegramBotHookConf
	}
	return hook, nil
}

func (t telegramBotHook) call(entry zapcore.Entry) error {
	msg := "⚠️Daemon hook⚠️\r\n" +
		"\r\nOrgID: " + config.GetString(config.OrganizationId) +
		"\r\nServiceID: " + config.GetString(config.ServiceId) +
		"\r\nLog Level: " + entry.Level.String() +
		"\r\nUTC Time: " + entry.Time.UTC().String() +
		"\r\nTime: " + entry.Time.String() +
		"\r\nFullPath: " + entry.Caller.FullPath() +
		"\r\nMessage: " + entry.Message +
		"\r\nStack: " + entry.Stack

	encoded, err := json.Marshal(telegramMessage{
		ChatID:              t.ChatID,
		Text:                msg,
		DisableNotification: t.DisableNotification,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(sendTGMessageURLTemplate, t.APIKey), bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request to Telegram API: %w", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			zap.L().Warn("can't close body", zap.Error(err))
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("response status code is not 200, it is %d", response.StatusCode)
	}

	if err := response.Body.Close(); err != nil {
		return err
	}
	return nil
}

func (t emailHook) call(entry zapcore.Entry) error {
	// Set up authentication information.
	auth := smtp.PlainAuth(t.applicationName, t.Username, t.Password, t.Host)

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	to := []string{t.To}
	msg := []byte("To: " + t.To +
		"\r\nSubject: Daemon hook!\r\n" +
		"\r\nOrgID: " + config.GetString(config.OrganizationId) +
		"\r\nServiceID: " + config.GetString(config.ServiceId) +
		"\r\nNetwork: " + config.GetString(config.BlockChainNetworkSelected) +
		"\r\nDaemon version: " + config.GetVersionTag() +
		"\r\nLog level: " + entry.Level.String() +
		"\r\nTime UTC: " + entry.Time.UTC().String() +
		"\r\nTime: " + entry.Time.String() +
		"\r\nFullPath: " + entry.Caller.FullPath() +
		"\r\nMessage: " + entry.Message +
		"\r\nStack: " + entry.Stack)

	err := smtp.SendMail(fmt.Sprintf("%s:%d", t.Host, t.Port), auth, t.From, to, msg)
	if err != nil {
		zap.L().Warn("can't send email via hook", zap.Error(err))
	}
	return err
}
