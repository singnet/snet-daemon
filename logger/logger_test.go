package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/singnet/snet-daemon/v6/config"
)

const defaultLogConfigJSON = `
	{
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "json",
			"timestamp_format": "2006-01-02T15:04:05.999999999Z07:00"
		},
		"output": {
			"type": "file",
			"file_pattern": "/tmp/snet-daemon.%Y%m%d.log",
			"current_link": "/tmp/snet-daemon.log",
			"max_size_in_mb": 86400,
			"max_age_in_days": 604800,
			"rotation_count": 0
		}
	}`

var vip *viper.Viper

func setupConfig() {
	vip = viper.New()
	vip.SetEnvPrefix("SNET")
	vip.AutomaticEnv()

	defaults := viper.New()
	err := config.ReadConfigFromJsonString(defaults, defaultLogConfigJSON)
	if err != nil {
		panic(fmt.Sprintf("Cannot load default config: %v", err))
	}
	config.SetDefaultFromConfig(vip, defaults)

	vip.AddConfigPath(".")

	config.SetVip(vip)
}

func TestMain(m *testing.M) {
	result := m.Run()

	removeLogFiles("/tmp/snet-daemon*.log")
	removeLogFiles("/tmp/file-rotatelogs-test.*.log")

	os.Exit(result)
}

func removeLogFiles(pattern string) {
	var err error
	var files []string

	files, err = filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("\nWarn:Cannot find files using pattern: %v", err)
	}

	for _, file := range files {
		err = os.Remove(file)
		if err != nil {
			fmt.Printf("\nWarn:Cannot remove file: %v, error: %v", file, err)
		}
	}
}

type testGetLocationTimezone struct {
	name          string
	timezone      string
	expectedError string
}

func TestGetLocationTimezone(t *testing.T) {
	setupConfig()

	testCases := []testGetLocationTimezone{
		{
			name:     "Valid timezone",
			timezone: "UTC",
		},
		{
			name:          "Invalid timezone",
			timezone:      "INVALID",
			expectedError: "unknown time zone INVALID",
		},
		{
			name:     "Valid timezone",
			timezone: "America/New_York",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vip.Set(LogTimezoneKey, tc.timezone)

			timezone, err := getLocationTimezone()

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				currentTime := time.Now()
				assert.Equal(t, currentTime.Format(tc.timezone), currentTime.Format(timezone.String()))
			}
		})
	}
}

type encoderConfigTestCase struct {
	name            string
	timeStampFormat string
	timezone        string
	expectedError   string
}

func TestCreateEncoderConfig(t *testing.T) {
	setupConfig()

	testCases := []encoderConfigTestCase{
		{
			name:            "Valid timestamp format",
			timeStampFormat: "2006-01-02",
			timezone:        "UTC",
		},
		{
			name:     "Default timestamp format",
			timezone: "UTC",
		},
		{
			name:          "Invalid timezone",
			timezone:      "INVALID",
			expectedError: "unknown time zone INVALID",
		},
		{
			name:     "Invalid timezone",
			timezone: "America/New_York",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vip.Set(LogTimezoneKey, tc.timezone)
			if tc.timeStampFormat != "" {
				vip.Set(LogTimestampFormatKey, tc.timeStampFormat)
			}

			encoderConfig, err := createEncoderConfig()

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encoderConfig)
			}
		})
	}
}

type loggerEncoderTestCases struct {
	name          string
	formatterType string
	expectedError string
}

func TestGetLoggerEncoder(t *testing.T) {
	setupConfig()

	testCases := []loggerEncoderTestCases{
		{
			name:          "Valid formatter type",
			formatterType: "text",
		},
		{
			name:          "Valid formatter type",
			formatterType: "json",
		},
		{
			name:          "Invalid formatter type",
			formatterType: "invalid",
			expectedError: "unsupported log formatter type: invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vip.Set(LogFormatterTypeKey, tc.formatterType)

			encoderConfig, err := createEncoderConfig()

			assert.NoError(t, err)
			assert.NotNil(t, encoderConfig)

			encoder, err := createEncoder(encoderConfig)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())

			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encoder)
			}

		})
	}
}

type logLevelTestCases struct {
	name          string
	inputLevel    string
	levelZap      zapcore.Level
	expectedError string
}

func TestGetLoggerLevel(t *testing.T) {
	testCases := []logLevelTestCases{
		{
			name:       "Valid log level",
			inputLevel: "debug",
			levelZap:   zap.DebugLevel,
		},
		{
			name:       "Valid log level",
			inputLevel: "info",
			levelZap:   zap.InfoLevel,
		},
		{
			name:       "Valid log level",
			inputLevel: "warn",
			levelZap:   zap.WarnLevel,
		},
		{
			name:       "Valid log level",
			inputLevel: "error",
			levelZap:   zap.ErrorLevel,
		},
		{
			name:       "Valid log level",
			inputLevel: "panic",
			levelZap:   zap.PanicLevel,
		},
		{
			name:          "Invalid log level",
			inputLevel:    "invalid",
			expectedError: "wrong string for level: invalid. Available options: debug, info, warn, error, panic",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logLevel, err := getLoggerLevel(tc.inputLevel)
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.levelZap, logLevel)
			}
		})
	}
}

type formatFileNameTestCases struct {
	name             string
	filePatternName  string
	expectedFileName string
	expectedError    string
}

func TestFormatFileName(t *testing.T) {
	mockTime := time.Date(2024, 7, 4, 12, 34, 56, 789000000, time.UTC)

	testCases := []formatFileNameTestCases{
		{
			name:             "Valid file pattern name",
			filePatternName:  "./snet-daemon.%Y-----%m-----%d--%M.log",
			expectedFileName: "./snet-daemon.2024-----07-----04--34.log",
		},
		{
			name:            "Invalid file pattern name",
			filePatternName: "./snet-daemon.%L-----%E-----%O--%A.log",
			expectedError:   "invalid placeholder found in pattern: %L",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileName, err := formatFileName(tc.filePatternName, mockTime)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedFileName, fileName)
			}
		})
	}
}

type createWriterSyncerTestCases struct {
	name            string
	outputType      any
	filePatternName string
	expectedError   string
}

func TestCreateWriterSyncer(t *testing.T) {
	setupConfig()

	testCases := []createWriterSyncerTestCases{
		{
			name:            "Valid single output type",
			outputType:      "file",
			filePatternName: "./snet-daemon.%Y%m%d.log",
		},
		{
			name:            "Valid multiple output types",
			outputType:      []string{"file", "stdout", "stderr"},
			filePatternName: "./snet-daemon.%Y%m%d%M.log",
		},
		{
			name:            "No output types",
			outputType:      "",
			filePatternName: "./snet-daemon.%Y%m%d%M.log",
			expectedError:   "failed to read log.output.type from config: []",
		},
		{
			name:            "Invalid single output type",
			outputType:      "invalid",
			filePatternName: "./snet-daemon.%Y%m%d%M.log",
			expectedError:   "unsupported log output type: invalid",
		},
		{
			name:            "Invalid multiple output types",
			outputType:      []string{"invalid1", "invalid2"},
			filePatternName: "./snet-daemon.%Y%m%d%M.log",
			expectedError:   "unsupported log output type: invalid1",
		},
		{
			name:            "Invalid file pattern name",
			outputType:      "file",
			filePatternName: "./snet-daemon.%L.log",
			expectedError:   "failed to create file writer for logger, invalid placeholder found in pattern: %L",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			vip.Set(LogOutputTypeKey, tc.outputType)
			vip.Set(LogOutputFilePatternKey, tc.filePatternName)
			ws, err := createWriterSyncer()

			if tc.expectedError != "" {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, ws)
			}
		})
	}
}

type configTestCase struct {
	name        string
	config      map[string]any
	expectPanic bool
	expectedLog *zap.Logger
}

func TestInitialize(t *testing.T) {
	// Setup default configuration once
	setupConfig()

	testCases := []configTestCase{
		{
			name: "Valid config",
			config: map[string]any{
				"log.level":                      "info",
				"log.timezone":                   "UTC",
				"log.formatter.type":             "json",
				"log.formatter.timestamp_format": "UTC",
				"log.output.type":                []string{"file", "stdout"},
				"log.output.file_pattern":        "/tmp/snet-daemon.%Y%m%d.log",
				"log.output.current_link":        "/tmp/snet-daemon.log",
				"log.output.max_size_in_mb":      86400,
				"log.output.max_age_in_days":     604800,
				"log.output.rotation_count":      0,
			},
			expectPanic: false,
			expectedLog: zap.L(),
		},
		{
			name: "Invalid config - invalid level",
			config: map[string]any{
				"log.level":                      "INVALID",
				"log.timezone":                   "UTC",
				"log.formatter.type":             "json",
				"log.formatter.timestamp_format": "UTC",
				"log.output.type":                "file",
				"log.output.file_pattern":        "/tmp/snet-daemon.%Y%m%d.log",
				"log.output.current_link":        "/tmp/snet-daemon.log",
				"log.output.max_size_in_mb":      86400,
				"log.output.max_age_in_days":     604800,
				"log.output.rotation_count":      0,
			},
			expectPanic: true,
		},
		{
			name: "Invalid config - invalid formatter type",
			config: map[string]any{
				"log.level":                      "info",
				"log.timezone":                   "UTC",
				"log.formatter.type":             "INVALID",
				"log.formatter.timestamp_format": "UTC",
				"log.output.type":                "file",
				"log.output.file_pattern":        "/tmp/snet-daemon.%Y%m%d.log",
				"log.output.current_link":        "/tmp/snet-daemon.log",
				"log.output.max_size_in_mb":      86400,
				"log.output.max_age_in_days":     604800,
				"log.output.rotation_count":      0,
			},
			expectPanic: true,
		},
		{
			name: "Invalid config - invalid output type",
			config: map[string]any{
				"log.level":                      "info",
				"log.timezone":                   "UTC",
				"log.formatter.type":             "json",
				"log.formatter.timestamp_format": "UTC",
				"log.output.type":                []string{"INVALID"},
				"log.output.file_pattern":        "/tmp/snet-daemon.%Y%m%d.log",
				"log.output.current_link":        "/tmp/snet-daemon.log",
				"log.output.max_size_in_mb":      86400,
				"log.output.max_age_in_days":     604800,
				"log.output.rotation_count":      0,
			},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the configuration for each test case
			for key, value := range tc.config {
				vip.Set(key, value)
			}

			if tc.expectPanic {
				assert.Panics(t, func() {
					Initialize()
				}, "expected panic but did not get one")
			} else {
				Initialize()
				assert.NotNil(t, zap.L(), "Logger should not be nil")
			}
		})
	}
}
