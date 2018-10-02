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

// Hook is a structure to return new log message hooks instances from factory
// method.
type Hook struct {
	// Delegate keeps logrus hook pointer to call on message.
	Delegate log.Hook
	// ExitHandler is a function to be called before terminating application
	// when logrus.Fatal() method was called.
	ExitHandler func()
}

// RegisterHookType registers new hook type in the system.
func RegisterHookType(hookType string, hookFactoryMethod func(*viper.Viper) (*Hook, error)) {
	hookFactoryMethodsByType[hookType] = hookFactoryMethod
}

var hookFactoryMethodsByType = map[string]func(*viper.Viper) (*Hook, error){}

func init() {
	RegisterHookType("mail_auth", newMailAuthHook)
}

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

	var hookFactoryMethod func(*viper.Viper) (*Hook, error)
	hookFactoryMethod, ok = hookFactoryMethodsByType[hookType]
	if !ok {
		return fmt.Errorf("Unexpected hook type: \"%v\"", hookType)
	}

	var hook *Hook
	hook, err = hookFactoryMethod(config.Sub(LogHookConfigKey))
	if err != nil {
		return fmt.Errorf("Cannot create hook instance: %v", err)
	}
	var internalHook = internalHook{delegate: hook.Delegate, exitHandler: hook.ExitHandler}

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
	internalHook.levels = levels

	logger.AddHook(&internalHook)
	log.RegisterExitHandler(internalHook.exitHandler)

	return nil
}

func newMailAuthHook(config *viper.Viper) (*Hook, error) {
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
	return &Hook{Delegate: mailAuthHook, ExitHandler: func() {}}, nil
}
