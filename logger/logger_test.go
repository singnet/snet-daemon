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

const (
	testConfigJSON string = `
{
    "json-formatter": {
        "type": "json",
        "timezone": "UTC"
    },
    "text-formatter": {
        "type": "text",
        "timezone": "UTC"
    },
    "incorrect-type-formatter": {
        "type": "UNKNOWN"
    },
    "incorrect-timezone-formatter": {
        "type": "text",
        "timezone": "UNKNOWN"
    },

    "file-output": {
        "type": "file",
        "file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
        "current_link": "/tmp/snet-daemon.log",
        "clock_timezone": "UTC",
        "rotation_time_in_sec": 86400,
        "max_age_in_sec": 604800,
        "rotation_count": 0
    },
    "stdout-output": {
        "type": "stdout"
    },
    "incorrect-type-output": {
        "type": "UNKNOWN"
    },
    "incorrect-timezone-output": {
        "type": "file",
        "clock_timezone": "UNKNOWN"
    },
    "incorrect-file-pattern-output": {
        "type": "file",
        "file_pattern": "%5"
    },

	"log":  {
		"level": "info",
		"formatter": {
			"type": "json",
			"timezone": "UTC"
		},
		"output": {
			"type": "file",
			"file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
			"current_link": "/tmp/snet-daemon.log",
			"clock_timezone": "UTC",
			"rotation_time_in_sec": 86400,
			"max_age_in_sec": 604800,
			"rotation_count": 0
		}
	},
	"incorrect-level-log":  {
		"level": "UNKNOWN"
	},
	"incorrect-formatter-log":  {
		"formatter": {
			"type": "UNKNOWN"
		}
	},
	"incorrect-output-log":  {
		"output": {
			"type": "UNKNOWN"
		}
	}
}
`
)

const defaultLogConfigJSON = `
	{
		"level": "info",
		"formatter": {
			"type": "json",
			"timezone": "UTC"
		},
		"output": {
			"type": "file",
			"file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
			"current_link": "/tmp/snet-daemon.log",
			"clock_timezone": "UTC",
			"rotation_time_in_sec": 86400,
			"max_age_in_sec": 604800,
			"rotation_count": 0
		}
	}`

var testConfig = readConfig(testConfigJSON)
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
	var formatterConfig = testConfig.Sub("text-formatter")

	var formatter, err = newFormatterByConfig(formatterConfig)

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.TextFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.TextFormatter{})
	assert.Equal(t, time.UTC, formatter.timestampLocation, "Incorrect timestampLocation")
}

func TestNewFormatterJsonType(t *testing.T) {
	var formatterConfig = testConfig.Sub("json-formatter")

	var formatter, err = newFormatterByConfig(formatterConfig)

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})

	assert.Equal(t, time.UTC, formatter.timestampLocation, "Incorrect timestampLocation")
}

func TestNewFormatterIncorrectType(t *testing.T) {
	var formatterConfig = testConfig.Sub("incorrect-type-formatter")
	config.SetDefaultFromConfig(formatterConfig, testConfig.Sub("json-formatter"))

	var _, err = newFormatterByConfig(formatterConfig)

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Unexpected formatter type: UNKNOWN"), err, "Unexpected error message")
}

func TestNewFormatterIncorrectTimestampTimezone(t *testing.T) {
	var formatterConfig = testConfig.Sub("incorrect-timezone-formatter")
	config.SetDefaultFromConfig(formatterConfig, testConfig.Sub("json-formatter"))

	var _, err = newFormatterByConfig(formatterConfig)

	assert.NotNil(t, err)
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
	config.SetDefaultFromConfig(formatterConfig, testConfig.Sub("json-formatter"))

	var formatter, err = newFormatterByConfig(formatterConfig)

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})

	assert.Equal(t, time.UTC, formatter.timestampLocation, "Incorrect timestampLocation")
}

func TestNewOutputFile(t *testing.T) {
	var outputConfig = testConfig.Sub("file-output")

	var writer, err = newOutputByConfig(outputConfig)

	assert.Nil(t, err)
	_, isFileWriter := writer.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", writer, &rotatelogs.RotateLogs{})
}

func TestNewOutputStdout(t *testing.T) {
	var outputConfig = testConfig.Sub("stdout-output")

	var writer, err = newOutputByConfig(outputConfig)

	assert.Nil(t, err)
	assert.Equal(t, os.Stdout, writer, "Unexpected writer type")
}

func TestNewOutputIncorrectType(t *testing.T) {
	var outputConfig = testConfig.Sub("incorrect-type-output")
	config.SetDefaultFromConfig(outputConfig, testConfig.Sub("file-output"))

	var _, err = newOutputByConfig(outputConfig)

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Unexpected output type: UNKNOWN"), err, "Unexpected error message")
}

func TestNewOutputIncorrectClockTimezone(t *testing.T) {
	var outputConfig = testConfig.Sub("incorrect-timezone-output")
	config.SetDefaultFromConfig(outputConfig, testConfig.Sub("file-output"))

	var _, err = newOutputByConfig(outputConfig)

	assert.NotNil(t, err)
}

func TestNewIncorrectFileOutputFilePattern(t *testing.T) {
	var outputConfig = testConfig.Sub("incorrect-file-pattern-output")
	config.SetDefaultFromConfig(outputConfig, testConfig.Sub("file-output"))

	var _, err = newOutputByConfig(outputConfig)

	assert.NotNil(t, err)
}

func TestNewOutputDefault(t *testing.T) {
	var outputConfig = viper.New()
	config.SetDefaultFromConfig(outputConfig, testConfig.Sub("file-output"))

	var writer, err = newOutputByConfig(outputConfig)

	assert.Nil(t, err)
	_, isFileWriter := writer.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", writer, &rotatelogs.RotateLogs{})
}

func TestInitLogger(t *testing.T) {
	var loggerConfig = testConfig.Sub("log")

	var err = InitLogger(loggerConfig)

	assert.Nil(t, err)
}

func TestInitLoggerIncorrectLevel(t *testing.T) {
	var loggerConfig = testConfig.Sub("incorrect-level-log")
	config.SetDefaultFromConfig(loggerConfig, testConfig.Sub("log"))

	var err = InitLogger(loggerConfig)

	assert.Equal(t, errors.New("Unable parse log level string: UNKNOWN, err: not a valid logrus Level: \"UNKNOWN\""), err, "Unexpected error message message")
}

func TestInitLoggerIncorrectFormatter(t *testing.T) {
	var loggerConfig = testConfig.Sub("incorrect-formatter-log")
	config.SetDefaultFromConfig(loggerConfig, testConfig.Sub("log"))

	var err = InitLogger(loggerConfig)

	assert.Equal(t, errors.New("Unable initialize log formatter, error: Unexpected formatter type: UNKNOWN"), err, "Unexpected error message")
}

func TestInitLoggerIncorrectOutput(t *testing.T) {
	var loggerConfig = testConfig.Sub("incorrect-output-log")
	config.SetDefaultFromConfig(loggerConfig, testConfig.Sub("log"))

	var err = InitLogger(loggerConfig)

	assert.Equal(t, errors.New("Unable initialize log output, error: Unexpected output type: UNKNOWN"), err, "Unexpected error message")
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
