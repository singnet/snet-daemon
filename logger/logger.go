package logger

import (
	"fmt"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

const (
	LogFormatterTypeKey     = "type"
	LogFormatterTimezoneKey = "timezone"
)

func InitLogger() error {
	var err error

	var logLevel log.Level
	var logLevelConfig = config.GetString(config.LogLevelKey)
	logLevel, err = log.ParseLevel(logLevelConfig)
	if err != nil {
		return fmt.Errorf("Unable parse log level value: %v, err: %v", logLevelConfig, err)
	}
	log.SetLevel(logLevel)

	var formatter log.Formatter
	formatter, err = newFormatterByConfig(config.Vip().Sub(config.LogFormatterKey))
	if err != nil {
		return fmt.Errorf("Unable initialize formatter, error: %v", err)
	}
	log.SetFormatter(formatter)

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
