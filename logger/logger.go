package logger

import (
	"errors"
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/zbindenren/logrus_mail"
	"io"
	"os"
	"time"
)

// Logger configuration keys
const (
	LogLevelKey     = "level"
	LogFormatterKey = "formatter"
	LogOutputKey    = "output"
	LogHooksKey     = "hooks"

	LogFormatterTypeKey     = "type"
	LogFormatterTimezoneKey = "timezone"

	LogOutputTypeKey                  = "type"
	LogOutputFileFilePatternKey       = "file_pattern"
	LogOutputFileCurrentLinkKey       = "current_link"
	LogOutputFileClockTimezoneKey     = "clock_timezone"
	LogOutputFileRotationTimeInSecKey = "rotation_time_in_sec"
	LogOutputFileMaxAgeInSecKey       = "max_age_in_sec"
	LogOutputFileRotationCountKey     = "rotation_count"

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

func InitLogger(config *viper.Viper) error {
	return initLogger(log.StandardLogger(), config)
}

// InitLogger initializes logger using configuration provided by viper
// instance.
//
// Function designed to configure few different loggers with different
// formatter and output settings. To achieve this viper configuration
// contains separate sections for each logger, each output and
// each formatter.
func initLogger(logger *log.Logger, config *viper.Viper) error {
	var err error

	var level log.Level
	var levelString = config.GetString(LogLevelKey)
	level, err = log.ParseLevel(levelString)
	if err != nil {
		return fmt.Errorf("Unable parse log level string: %v, err: %v", levelString, err)
	}
	logger.SetLevel(level)

	var formatter log.Formatter
	formatter, err = newFormatterByConfig(config.Sub(LogFormatterKey))
	if err != nil {
		return fmt.Errorf("Unable initialize log formatter, error: %v", err)
	}
	logger.SetFormatter(formatter)

	var output io.Writer
	output, err = newOutputByConfig(config.Sub(LogOutputKey))
	if err != nil {
		return fmt.Errorf("Unable initialize log output, error: %v", err)
	}
	logger.SetOutput(output)

	for _, hookConfigName := range config.GetStringSlice(LogHooksKey) {
		err = addHookByConfig(logger, config.Sub(hookConfigName))
		if err != nil {
			return fmt.Errorf("Unable to add log hook \"%v\", error: %v", hookConfigName, err)
		}
	}

	logger.Info("Logger initialized")

	return nil
}

func newFormatterByConfig(config *viper.Viper) (*timezoneFormatter, error) {
	var err error
	var formatter = &timezoneFormatter{}

	switch formatterType := config.GetString(LogFormatterTypeKey); formatterType {
	case "text":
		formatter.delegate = &log.TextFormatter{}
	case "json":
		formatter.delegate = &log.JSONFormatter{}
	default:
		return nil, fmt.Errorf("Unexpected formatter type: %v", formatterType)
	}

	var location *time.Location
	location, err = time.LoadLocation(config.GetString(LogFormatterTimezoneKey))
	if err != nil {
		return nil, err
	}
	formatter.timestampLocation = location

	return formatter, nil
}

type timezoneFormatter struct {
	delegate          log.Formatter
	timestampLocation *time.Location
}

func (formatter *timezoneFormatter) Format(entry *log.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(formatter.timestampLocation)
	return formatter.delegate.Format(entry)
}

func newOutputByConfig(config *viper.Viper) (io.Writer, error) {
	var err error

	switch outputType := config.GetString(LogOutputTypeKey); outputType {
	case "file":

		var location *time.Location
		if location, err = time.LoadLocation(config.GetString(LogOutputFileClockTimezoneKey)); err != nil {
			return nil, err
		}

		var fileWriter io.Writer
		fileWriter, err = rotatelogs.New(config.GetString(LogOutputFileFilePatternKey),
			rotatelogs.WithLocation(location),
			rotatelogs.WithLinkName(config.GetString(LogOutputFileCurrentLinkKey)),
			rotatelogs.WithRotationTime(config.GetDuration(LogOutputFileRotationTimeInSecKey)*time.Second),
			rotatelogs.WithMaxAge(config.GetDuration(LogOutputFileMaxAgeInSecKey)*time.Second),
			rotatelogs.WithRotationCount(uint(config.GetInt(LogOutputFileRotationCountKey))),
		)
		if err != nil {
			return nil, err
		}

		return fileWriter, nil
	case "stdout":
		return os.Stdout, nil
	default:
		return nil, fmt.Errorf("Unexpected output type: %v", outputType)
	}

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
