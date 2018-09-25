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

var testConfig *viper.Viper

func TestMain(m *testing.M) {
	testConfig = readConfig(testConfigJSON)

	result := m.Run()

	removeLogFiles()

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

func removeLogFiles() {
	var err error
	var files []string

	files, err = filepath.Glob("/tmp/snet-daemon*.log")
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

func TestNewFormatterTextType(t *testing.T) {
	var formatterConfig = testConfig.Sub("text-formatter")

	var formatter, err = newFormatterByConfig(&configWithDefaults{formatterConfig, viper.New()})

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.TextFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.TextFormatter{})
	assert.Equal(t, time.UTC, formatter.timestampLocation, "Incorrect timestampLocation")
}

func TestNewFormatterJsonType(t *testing.T) {
	var formatterConfig = testConfig.Sub("json-formatter")

	var formatter, err = newFormatterByConfig(&configWithDefaults{formatterConfig, viper.New()})

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})

	assert.Equal(t, time.UTC, formatter.timestampLocation, "Incorrect timestampLocation")
}

func TestNewFormatterIncorrectType(t *testing.T) {
	var defaultConfig = testConfig.Sub("json-formatter")
	var formatterConfig = testConfig.Sub("incorrect-type-formatter")

	var _, err = newFormatterByConfig(&configWithDefaults{formatterConfig, defaultConfig})

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Unexpected formatter type: UNKNOWN"), err, "Unexpected error message")
}

func TestNewFormatterIncorrectTimestampTimezone(t *testing.T) {
	var defaultConfig = testConfig.Sub("json-formatter")
	var formatterConfig = testConfig.Sub("incorrect-timezone-formatter")

	var _, err = newFormatterByConfig(&configWithDefaults{formatterConfig, defaultConfig})

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
	var defaultConfig = testConfig.Sub("json-formatter")

	var formatter, err = newFormatterByConfig(&configWithDefaults{nil, defaultConfig})

	assert.Nil(t, err)
	_, isFormatterDelegate := formatter.delegate.(*log.JSONFormatter)
	assert.True(t, isFormatterDelegate, "Unexpected underlying formatter type, actual: %T, expected: %T", formatter.delegate, &log.JSONFormatter{})

	assert.Equal(t, time.UTC, formatter.timestampLocation, "Incorrect timestampLocation")
}

func TestNewOutputFile(t *testing.T) {
	var outputConfig = testConfig.Sub("file-output")

	var writer, err = newOutputByConfig(&configWithDefaults{outputConfig, viper.New()})

	assert.Nil(t, err)
	_, isFileWriter := writer.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", writer, &rotatelogs.RotateLogs{})
}

func TestNewOutputStdout(t *testing.T) {
	var outputConfig = testConfig.Sub("stdout-output")

	var writer, err = newOutputByConfig(&configWithDefaults{outputConfig, viper.New()})

	assert.Nil(t, err)
	assert.Equal(t, os.Stdout, writer, "Unexpected writer type")
}

func TestNewOutputIncorrectType(t *testing.T) {
	var defaultConfig = testConfig.Sub("file-output")
	var outputConfig = testConfig.Sub("incorrect-type-output")

	var _, err = newOutputByConfig(&configWithDefaults{outputConfig, defaultConfig})

	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Unexpected output type: UNKNOWN"), err, "Unexpected error message")
}

func TestNewOutputIncorrectClockTimezone(t *testing.T) {
	var defaultConfig = testConfig.Sub("file-output")
	var outputConfig = testConfig.Sub("incorrect-timezone-output")

	var _, err = newOutputByConfig(&configWithDefaults{outputConfig, defaultConfig})

	assert.NotNil(t, err)
}

func TestNewIncorrectFileOutputFilePattern(t *testing.T) {
	var defaultConfig = testConfig.Sub("file-output")
	var outputConfig = testConfig.Sub("incorrect-file-pattern-output")

	var _, err = newOutputByConfig(&configWithDefaults{outputConfig, defaultConfig})

	assert.NotNil(t, err)
}

func TestNewOutputDefault(t *testing.T) {
	var defaultConfig = testConfig.Sub("file-output")

	var writer, err = newOutputByConfig(&configWithDefaults{nil, defaultConfig})

	assert.Nil(t, err)
	_, isFileWriter := writer.(*rotatelogs.RotateLogs)
	assert.True(t, isFileWriter, "Unexpected writer type, actual: %T, expected: %T", writer, &rotatelogs.RotateLogs{})
}

func TestInitLogger(t *testing.T) {
	var loggerConfig = testConfig.Sub("log")

	var err = InitLogger(loggerConfig, viper.New())

	assert.Nil(t, err)
}

func TestInitLoggerIncorrectLevel(t *testing.T) {
	var defaultConfig = testConfig.Sub("log")
	var loggerConfig = testConfig.Sub("incorrect-level-log")

	var err = InitLogger(loggerConfig, defaultConfig)

	assert.Equal(t, errors.New("Unable parse log level string: UNKNOWN, err: not a valid logrus Level: \"UNKNOWN\""), err, "Unexpected error message message")
}

func TestInitLoggerIncorrectFormatter(t *testing.T) {
	var defaultConfig = testConfig.Sub("log")
	var loggerConfig = testConfig.Sub("incorrect-formatter-log")

	var err = InitLogger(loggerConfig, defaultConfig)

	assert.Equal(t, errors.New("Unable initialize log formatter, error: Unexpected formatter type: UNKNOWN"), err, "Unexpected error message")
}

func TestInitLoggerIncorrectOutput(t *testing.T) {
	var defaultConfig = testConfig.Sub("log")
	var loggerConfig = testConfig.Sub("incorrect-output-log")

	var err = InitLogger(loggerConfig, defaultConfig)

	assert.Equal(t, errors.New("Unable initialize log output, error: Unexpected output type: UNKNOWN"), err, "Unexpected error message")
}
