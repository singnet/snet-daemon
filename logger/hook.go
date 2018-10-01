package logger

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/zbindenren/logrus_mail"
)

const (
	LogHookTypeKey   = "type"
	LogHookLevelsKey = "levels"
	LogHookConfigKey = "config"

	LogHookMailApplicationNameKey = "application_name"
	LogHookMailHostKey            = "host"
	LogHookMailPortKey            = "port"
	LogHookMailFromKey            = "from"
	LogHookMailToKey              = "to"
	LogHookMailUsernameKey        = "username"
	LogHookMailPasswordKey        = "password"
)

type internalHook struct {
	delegate    log.Hook
	exitHandler func()
	levels      []log.Level
}

func (hook *internalHook) Fire(entry *log.Entry) error {
	return hook.delegate.Fire(entry)
}

func (hook *internalHook) Levels() []log.Level {
	return hook.levels
}

var hookFactoryMethodsByType = map[string]func(*viper.Viper) (*internalHook, error){
	"mail_auth": newMailAuthHook,
}

func addHookByConfig(logger *log.Logger, config *viper.Viper) error {
	var err error
	var ok bool

	if config == nil {
		return errors.New("No hook definition")
	}

	var hookType = config.GetString(LogHookTypeKey)
	if hookType == "" {
		return errors.New("No hook type in hook config")
	}

	var hookFactoryMethod func(*viper.Viper) (*internalHook, error)
	hookFactoryMethod, ok = hookFactoryMethodsByType[hookType]
	if !ok {
		return fmt.Errorf("Unexpected hook type: \"%v\"", hookType)
	}

	var hook *internalHook
	hook, err = hookFactoryMethod(config.Sub(LogHookConfigKey))
	if err != nil {
		return fmt.Errorf("Cannot create hook instance: %v", err)
	}

	if config.Get(LogHookLevelsKey) == nil {
		return errors.New("No levels in hook config")
	}
	var levels []log.Level
	for _, levelString := range config.GetStringSlice(LogHookLevelsKey) {
		var level log.Level
		level, err = log.ParseLevel(levelString)
		if err != nil {
			return fmt.Errorf("Unable parse log level string: \"%v\", err: %v", levelString, err)
		}
		levels = append(levels, level)
	}
	hook.levels = levels

	logger.AddHook(hook)
	log.RegisterExitHandler(hook.exitHandler)

	return nil
}

func newMailAuthHook(config *viper.Viper) (*internalHook, error) {
	if config == nil {
		return nil, errors.New("Unable to create instance of mail auth hook: no config provided")
	}
	var mailAuthHook, err = logrus_mail.NewMailAuthHook(
		config.GetString(LogHookMailApplicationNameKey),
		config.GetString(LogHookMailHostKey),
		config.GetInt(LogHookMailPortKey),
		config.GetString(LogHookMailFromKey),
		config.GetString(LogHookMailToKey),
		config.GetString(LogHookMailUsernameKey),
		config.GetString(LogHookMailPasswordKey),
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to create instance of mail auth hook: %v", err)
	}
	return &internalHook{delegate: mailAuthHook, exitHandler: func() {}}, nil
}
