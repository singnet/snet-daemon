package logger

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"time"
)

const (
	LogFormatterTypeKey            = "type"
	LogFormatterTimezoneKey        = "timezone"
	LogOutputTypeKey               = "type"
	LogOutputFileFilePattern       = "file_pattern"
	LogOutputFileCurrentLink       = "current_link"
	LogOutputFileClockTimezone     = "clock_timezone"
	LogOutputFileRotationTimeInSec = "rotation_time_in_sec"
	LogOutputFileMaxAgeInSec       = "max_age_in_sec"
	LogOutputFileRotationCount     = "rotation_count"
)

func InitLogger() error {
	var err error

	var level log.Level
	var levelString = config.GetString(config.LogLevelKey)
	level, err = log.ParseLevel(levelString)
	if err != nil {
		return fmt.Errorf("Unable parse log level string: %v, err: %v", levelString, err)
	}
	log.SetLevel(level)

	var formatter log.Formatter
	formatter, err = newFormatterByConfig(config.Vip().Sub(config.LogFormatterKey))
	if err != nil {
		return fmt.Errorf("Unable initialize log formatter, error: %v", err)
	}
	log.SetFormatter(formatter)

	var output io.Writer
	output, err = newOutputByConfig(config.Vip().Sub(config.LogOutputKey))
	if err != nil {
		return fmt.Errorf("Unable initialize log output, error: %v, err")
	}
	log.SetOutput(output)

	return nil
}

func newFormatterByConfig(config *viper.Viper) (*timezoneFormatter, error) {
	var err error
	var formatter *timezoneFormatter = &timezoneFormatter{}

	config.SetDefault(LogFormatterTypeKey, "json")
	config.SetDefault(LogFormatterTimezoneKey, time.Local.String())

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

	config.SetDefault(LogOutputTypeKey, "file")

	switch outputType := config.GetString(LogOutputTypeKey); outputType {
	case "file":

		config.SetDefault(LogOutputFileFilePattern, "./snet-daemon.%Y%m%d.metrics.log")
		config.SetDefault(LogOutputFileClockTimezone, time.Local.String())
		config.SetDefault(LogOutputFileCurrentLink, "")
		config.SetDefault(LogOutputFileRotationTimeInSec, 86400)
		config.SetDefault(LogOutputFileMaxAgeInSec, 604800)
		config.SetDefault(LogOutputFileRotationCount, 0)

		var location *time.Location
		if location, err = time.LoadLocation(config.GetString(LogOutputFileClockTimezone)); err != nil {
			return nil, err
		}

		var fileWriter io.Writer
		fileWriter, err = rotatelogs.New(config.GetString(LogOutputFileFilePattern),
			rotatelogs.WithLocation(location),
			rotatelogs.WithLinkName(config.GetString(LogOutputFileCurrentLink)),
			rotatelogs.WithRotationTime(config.GetDuration(LogOutputFileRotationTimeInSec)),
			rotatelogs.WithMaxAge(config.GetDuration(LogOutputFileMaxAgeInSec)),
			rotatelogs.WithRotationCount(uint(config.GetInt(LogOutputFileRotationCount))),
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
