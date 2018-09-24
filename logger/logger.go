package logger

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"io"
	"os"
	"time"
)

const (
	LogLevelKey     = "level"
	LogFormatterKey = "formatter"
	LogOutputKey    = "output"

	LogFormatterTypeKey     = "type"
	LogFormatterTimezoneKey = "timezone"

	LogOutputTypeKey                  = "type"
	LogOutputFileFilePatternKey       = "file_pattern"
	LogOutputFileCurrentLinkKey       = "current_link"
	LogOutputFileClockTimezoneKey     = "clock_timezone"
	LogOutputFileRotationTimeInSecKey = "rotation_time_in_sec"
	LogOutputFileMaxAgeInSecKey       = "max_age_in_sec"
	LogOutputFileRotationCountKey     = "rotation_count"
)

// InitLogger initializes logger using configuration provided by viper
// instance.
//
// Function designed to configure few different loggers with different
// formatter and output settings. To achieve this viper configuration
// contains separate sections for each logger, each output and
// each formatter. As viper doesn't support defaults for sub-configurations
// there is a workaround for this. Two different viper configurations should be
// provided: one for main configuration and one for configuration defaults.
func InitLogger(config *viper.Viper, defaults *viper.Viper) error {
	return initLogger(&configWithDefaults{config, defaults})
}

type configWithDefaults struct {
	config   *viper.Viper
	defaults *viper.Viper
}

func (config *configWithDefaults) get(key string) interface{} {
	if config.config != nil && config.config.InConfig(key) {
		return config.config.Get(key)
	} else {
		return config.defaults.Get(key)
	}
}

func (config *configWithDefaults) getString(key string) string {
	return cast.ToString(config.get(key))
}

func (config *configWithDefaults) getInt(key string) int {
	return cast.ToInt(config.get(key))
}

func (config *configWithDefaults) getDuration(key string) time.Duration {
	return cast.ToDuration(config.get(key))
}

func (config *configWithDefaults) sub(key string) *configWithDefaults {
	return &configWithDefaults{config.config.Sub(key), config.defaults.Sub(key)}
}

func initLogger(config *configWithDefaults) error {
	var err error

	var level log.Level
	var levelString = config.getString(LogLevelKey)
	level, err = log.ParseLevel(levelString)
	if err != nil {
		return fmt.Errorf("Unable parse log level string: %v, err: %v", levelString, err)
	}
	log.SetLevel(level)

	var formatter log.Formatter
	formatter, err = newFormatterByConfig(config.sub(LogFormatterKey))
	if err != nil {
		return fmt.Errorf("Unable initialize log formatter, error: %v", err)
	}
	log.SetFormatter(formatter)

	var output io.Writer
	output, err = newOutputByConfig(config.sub(LogOutputKey))
	if err != nil {
		return fmt.Errorf("Unable initialize log output, error: %v, err")
	}
	log.SetOutput(output)

	log.Info("Logger initialized")

	return nil
}

func newFormatterByConfig(config *configWithDefaults) (*timezoneFormatter, error) {
	var err error
	var formatter *timezoneFormatter = &timezoneFormatter{}

	switch formatterType := config.getString(LogFormatterTypeKey); formatterType {
	case "text":
		formatter.delegate = &log.TextFormatter{}
	case "json":
		formatter.delegate = &log.JSONFormatter{}
	default:
		return nil, fmt.Errorf("Unexpected formatter type: %v", formatterType)
	}

	var location *time.Location
	location, err = time.LoadLocation(config.getString(LogFormatterTimezoneKey))
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

func newOutputByConfig(config *configWithDefaults) (io.Writer, error) {
	var err error

	switch outputType := config.getString(LogOutputTypeKey); outputType {
	case "file":

		var location *time.Location
		if location, err = time.LoadLocation(config.getString(LogOutputFileClockTimezoneKey)); err != nil {
			return nil, err
		}

		var fileWriter io.Writer
		fileWriter, err = rotatelogs.New(config.getString(LogOutputFileFilePatternKey),
			rotatelogs.WithLocation(location),
			rotatelogs.WithLinkName(config.getString(LogOutputFileCurrentLinkKey)),
			rotatelogs.WithRotationTime(config.getDuration(LogOutputFileRotationTimeInSecKey)),
			rotatelogs.WithMaxAge(config.getDuration(LogOutputFileMaxAgeInSecKey)),
			rotatelogs.WithRotationCount(uint(config.getInt(LogOutputFileRotationCountKey))),
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
