package logger

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jonboulle/clockwork"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const defaultFormatterConfigJSON = `
	{
		"type": "json",
		"timezone": "UTC"
	}`

const defaultOutputConfigJSON = `
	{
		"type": "file",
		"file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
		"current_link": "/tmp/snet-daemon.log",
		"clock_timezone": "UTC",
		"rotation_time_in_sec": 86400,
		"max_age_in_sec": 604800,
		"rotation_count": 0
	}`
const defaultLogConfigJSON = `
	{
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "json"
		},
		"output": {
			"type": "file",
			"file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
			"current_link": "/tmp/snet-daemon.log",
			"rotation_time_in_sec": 86400,
			"max_age_in_sec": 604800,
			"rotation_count": 0
		}
	}`

var defaultFormatterConfig = readConfig(defaultFormatterConfigJSON)
var defaultOutputConfig = readConfig(defaultOutputConfigJSON)
var defaultLogConfig = readConfig(defaultLogConfigJSON)

func TestMain(m *testing.M) {
	result := m.Run()

	removeLogFiles("/tmp/snet-daemon*.log")
	removeLogFiles("/tmp/file-rotatelogs-test.*.log")

	os.Exit(result)
}

func readConfig(configJSON string) *viper.Viper {
	var err error

	var vip *viper.Viper = viper.New()
	err = config.ReadConfigFromJsonString(vip, configJSON)
	if err != nil {
		panic(fmt.Sprintf("Cannot read test config: %v", err))
	}

	return vip
}

func removeLogFiles(pattern string) {
	var err error
	var files []string

	files, err = filepath.Glob(pattern)
	if err != nil {
		panic(fmt.Sprintf("Cannot find files using pattern: %v", err))
	}

	for _, file := range files {
		err = os.Remove(file)
		if err != nil {
			panic(fmt.Sprintf("Cannot remove file: %v, error: %v", file, err))
		}
	}
}

func newConfigFromString(configString string, defaultVip *viper.Viper) *viper.Viper {
	var err error
	var configVip = viper.New()

	err = config.ReadConfigFromJsonString(configVip, configString)
	if err != nil {
		panic(fmt.Sprintf("Cannot read test config: %v", configString))
	}

	if defaultVip != nil {
		config.SetDefaultFromConfig(configVip, defaultVip)
	}

	return configVip
}

func TestNewFormatterTextType(t *testing.T) {
	var formatterJSON = `{
        "type": "text",
        "timezone": "Local"
    }`
	var formatterConfig = newConfigFromString(formatterJSON, nil)

	var formatter, err = newFormatterByConfig(formatterConfig)

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.TextFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.TextFormatter{})
	assert.Equal(t, time.Local, formatter.timestampLocation)
}

func TestNewFormatterJsonType(t *testing.T) {
	var formatterJSON = `{
        "type": "json",
        "timezone": "Local"
    }`
	var formatterConfig = newConfigFromString(formatterJSON, nil)

	var formatter, err = newFormatterByConfig(formatterConfig)

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})
	assert.Equal(t, time.Local, formatter.timestampLocation)
}

func TestNewFormatterIncorrectType(t *testing.T) {
	var formatterJSON = `{
        "type": "UNKNOWN"
    }`
	var formatterConfig = newConfigFromString(formatterJSON, defaultFormatterConfig)

	var _, err = newFormatterByConfig(formatterConfig)

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Unexpected formatter type: UNKNOWN"), err)
}

func TestNewFormatterIncorrectTimestampTimezone(t *testing.T) {
	var formatterJSON = `{
        "timezone": "UNKNOWN..."
    }`
	var formatterConfig = newConfigFromString(formatterJSON, defaultFormatterConfig)

	var _, err = newFormatterByConfig(formatterConfig)

	assert.Equal(t, errors.New("time: invalid location name"), err)
}

type underlyingFormatterMock struct {
}

func (f *underlyingFormatterMock) Format(entry *log.Entry) ([]byte, error) {
	var buffer *bytes.Buffer = &bytes.Buffer{}
	if _, err := buffer.WriteString(entry.Time.Format(time.RFC3339)); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func TestTimezoneFormatter(t *testing.T) {
	var clockMock clockwork.FakeClock
	if japanTimezone, err := time.LoadLocation("Asia/Tokyo"); err == nil {
		clockMock = clockwork.NewFakeClockAt(time.Date(1980, time.August, 3, 13, 0, 0, 0, japanTimezone))
	} else {
		t.Fatal("Cannot not get Japan timezone", err)
	}
	var formatter = timezoneFormatter{&underlyingFormatterMock{}, time.UTC}
	var timestamp = clockMock.Now()

	var formattedTimestamp, err = formatter.Format(&log.Entry{Time: timestamp})

	assert.Nil(t, err)
	assert.Equal(t, timestamp.In(time.UTC).Format(time.RFC3339), string(formattedTimestamp))
}

func TestNewFormatterDefault(t *testing.T) {
	var formatterConfig = viper.New()
	config.SetDefaultFromConfig(formatterConfig, defaultFormatterConfig)

	var formatter, err = newFormatterByConfig(formatterConfig)

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})
	assert.Equal(t, time.UTC, formatter.timestampLocation)
}

func TestNewOutputFile(t *testing.T) {
	var outputConfigJSON = `{
        "type": "file",
        "file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
        "current_link": "/tmp/snet-daemon.log",
        "clock_timezone": "Local",
        "rotation_time_in_sec": 86400,
        "max_age_in_sec": 604800,
        "rotation_count": 0
    }`
	var outputConfig = newConfigFromString(outputConfigJSON, nil)

	var writer, err = newOutputByConfig(outputConfig)

	assert.Nil(t, err)
	_, isFileWriter := writer.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", writer, &rotatelogs.RotateLogs{})
	// there is no simple way to check that other parameters are applied
	// correctly
}

func TestNewOutputStdout(t *testing.T) {
	var outputConfigJSON = `{
        "type": "stdout"
    }`
	var outputConfig = newConfigFromString(outputConfigJSON, nil)

	var writer, err = newOutputByConfig(outputConfig)

	assert.Nil(t, err)
	assert.Equal(t, os.Stdout, writer, "Unexpected writer type")
}

func TestNewOutputIncorrectType(t *testing.T) {
	var outputConfigJSON = `{
        "type": "UNKNOWN"
    }`
	var outputConfig = newConfigFromString(outputConfigJSON, defaultOutputConfig)

	var _, err = newOutputByConfig(outputConfig)

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Unexpected output type: UNKNOWN"), err)
}

func TestNewOutputIncorrectClockTimezone(t *testing.T) {
	var outputConfigJSON = `{
        "type": "file",
        "clock_timezone": "UNKNOWN..."
    }`
	var outputConfig = newConfigFromString(outputConfigJSON, defaultOutputConfig)

	var _, err = newOutputByConfig(outputConfig)

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("time: invalid location name"), err)
}

func TestNewIncorrectFileOutputFilePattern(t *testing.T) {
	var outputConfigJSON = `{
        "type": "file",
        "file_pattern": "%5"
    }`
	var outputConfig = newConfigFromString(outputConfigJSON, defaultOutputConfig)

	var _, err = newOutputByConfig(outputConfig)

	assert.NotNil(t, err)
}

func TestNewOutputDefault(t *testing.T) {
	var outputConfig = viper.New()
	config.SetDefaultFromConfig(outputConfig, defaultOutputConfig)

	var writer, err = newOutputByConfig(outputConfig)

	assert.Nil(t, err)
	_, isFileWriter := writer.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", writer, &rotatelogs.RotateLogs{})
}

func TestInitLogger(t *testing.T) {
	var loggerConfigJSON = `
	{
		"level": "error",
		"timezone": "Local",
		"formatter": {
			"type": "json"
		},
		"output": {
			"type": "file",
			"file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
			"current_link": "/tmp/snet-daemon.log",
			"rotation_time_in_sec": 86400,
			"max_age_in_sec": 604800,
			"rotation_count": 0
		}
	}`
	var loggerConfig = newConfigFromString(loggerConfigJSON, nil)
	var logger = log.New()

	var err = initLogger(logger, loggerConfig)

	assert.Nil(t, err)
	assert.Equal(t, log.ErrorLevel, logger.Level)
	var formatter = logger.Formatter.(*timezoneFormatter)
	assert.Equal(t, time.Local, formatter.timestampLocation)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})
	_, isFileWriter := logger.Out.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", logger.Out, &rotatelogs.RotateLogs{})
}

func TestInitLoggerIncorrectLevel(t *testing.T) {
	var loggerConfigJSON = `{
		"level": "UNKNOWN"
    }`
	var loggerConfig = newConfigFromString(loggerConfigJSON, defaultLogConfig)

	var err = InitLogger(loggerConfig)

	assert.Equal(t, errors.New("Unable parse log level string: UNKNOWN, err: not a valid logrus Level: \"UNKNOWN\""), err, "Unexpected error message message")
}

func TestInitLoggerIncorrectFormatter(t *testing.T) {
	var loggerConfigJSON = `{
		"formatter": {
			"type": "UNKNOWN"
		}
    }`
	var loggerConfig = newConfigFromString(loggerConfigJSON, defaultLogConfig)

	var err = InitLogger(loggerConfig)

	assert.Equal(t, errors.New("Unable initialize log formatter, error: Unexpected formatter type: UNKNOWN"), err, "Unexpected error message")
}

func TestInitLoggerIncorrectOutput(t *testing.T) {
	var loggerConfigJSON = `{
		"output": {
			"type": "UNKNOWN"
		}
    }`
	var loggerConfig = newConfigFromString(loggerConfigJSON, defaultLogConfig)

	var err = InitLogger(loggerConfig)

	assert.Equal(t, errors.New("Unable initialize log output, error: Unexpected output type: UNKNOWN"), err, "Unexpected error message")
}

func TestInitLoggerIncorrectTimezone(t *testing.T) {
	var loggerConfigJSON = `{
		"timezone": "UNKNOWN..."
    }`
	var loggerConfig = newConfigFromString(loggerConfigJSON, defaultLogConfig)
	var logger = log.New()

	var err = initLogger(logger, loggerConfig)

	assert.Equal(t, errors.New("Unable initialize log formatter, error: time: invalid location name"), err)
}

func TestLogRotation(t *testing.T) {
	var err error
	var logFileInfo os.FileInfo
	var startTime time.Time

	var clockMock = clockwork.NewFakeClockAt(
		time.Date(2018, time.September, 26, 13, 0, 0, 0, time.UTC))
	var logWriter, _ = rotatelogs.New(
		"/tmp/file-rotatelogs-test.%Y%m%d.log",
		rotatelogs.WithClock(clockMock),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(7*24*time.Hour),
	)
	var logger = log.New()
	logger.SetOutput(logWriter)

	// without Add(-time.Second) testStart time is surprisingly after log file
	// modification date. Difference is about few milliseconds
	startTime = time.Now().Add(-time.Second)
	logger.Info("mockedTime: ", clockMock.Now(), " realTime: ", time.Now())
	logFileInfo, err = os.Stat("/tmp/file-rotatelogs-test.20180926.log")
	assert.Nil(t, err, "Cannot read log file info")
	assert.Truef(t, logFileInfo.ModTime().After(startTime), "Log was not updated, test started at: %v, file updated at: %v", startTime, logFileInfo.ModTime())

	startTime = time.Now().Add(-time.Second)
	clockMock.Advance(10 * time.Hour)
	logger.Info("mockedTime: ", clockMock.Now(), " realTime: ", time.Now())
	logFileInfo, err = os.Stat("/tmp/file-rotatelogs-test.20180927.log")
	assert.NotNil(t, err, "Log was rotated before correct time")

	startTime = time.Now().Add(-time.Second)
	clockMock.Advance(1*time.Hour + 1*time.Second)
	logger.Info("mockedTime: ", clockMock.Now(), " realTime: ", time.Now())
	logFileInfo, err = os.Stat("/tmp/file-rotatelogs-test.20180927.log")
	assert.Nil(t, err, "Cannot read log file info")
	assert.Truef(t, logFileInfo.ModTime().After(startTime), "Log was not updated, test started at: %v, file updated at: %v", startTime, logFileInfo.ModTime())
}
