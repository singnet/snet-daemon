package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestEncoderFormatsTimeAndFields(t *testing.T) {
	for _, tc := range []struct{ name, layout, zone, wantTime string }{
		{"default UTC", "", "UTC", "2024-01-02T03:04:05Z"},
		{"custom UTC", "2006/01/02 15:04", "UTC", "2024/01/02 03:04"},
		{"local offset", "", "America/New_York", "2024-01-01T22:04:05-05:00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupConfig(t)
			vip.Set(LogTimestampFormatKey, tc.layout)
			vip.Set(LogTimezoneKey, tc.zone)
			vip.Set(LogOutputTypeKey, []string{"file"})
			cfg, err := createEncoderConfig()
			require.NoError(t, err)
			encoder, err := createEncoder(cfg)
			require.NoError(t, err)
			encoded, err := encoder.EncodeEntry(zapcore.Entry{
				Time: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), Level: zap.WarnLevel,
				Message: "test message", Caller: zapcore.EntryCaller{Defined: true, File: "/src/daemon.go", Line: 42},
			}, []zap.Field{zap.Duration("elapsed", 1500*time.Millisecond), zap.String("request", "abc")})
			require.NoError(t, err)
			defer encoded.Free()
			var fields map[string]any
			require.NoError(t, json.Unmarshal(encoded.Bytes(), &fields))
			require.Equal(t, tc.wantTime, fields["ts"])
			require.Equal(t, "WARN", fields["level"])
			require.Equal(t, "/src/daemon.go:42", fields["caller"])
			require.Equal(t, "test message", fields["msg"])
			require.Equal(t, "1.5s", fields["elapsed"])
			require.Equal(t, "abc", fields["request"])
		})
	}
}

func TestAdditionalLogLevels(t *testing.T) {
	for name, want := range map[string]zapcore.Level{"warning": zap.WarnLevel, "fatal": zap.FatalLevel} {
		level, err := getLoggerLevel(name)
		require.NoError(t, err)
		require.Equal(t, want, level)
	}
}

func TestWriterCurrentLink(t *testing.T) {
	setupConfig(t)
	dir := t.TempDir()
	probe := filepath.Join(dir, "probe")
	if err := os.Symlink(filepath.Join(dir, "target"), probe); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("Windows does not permit symlinks: %v", err)
		}
		require.NoError(t, err)
	}
	require.NoError(t, os.Remove(probe))
	link := filepath.Join(dir, "current.log")
	vip.Set(LogOutputTypeKey, []string{"file"})
	vip.Set(LogOutputCurrentLinkKey, link)
	for _, name := range []string{"first.log", "second.log"} {
		target := filepath.Join(dir, name)
		vip.Set(LogOutputFilePatternKey, target)
		writer, err := createWriterSyncer()
		require.NoError(t, err)
		require.NotNil(t, writer)
		actual, err := os.Readlink(link)
		require.NoError(t, err)
		require.Equal(t, target, actual)
	}
}

func TestWriterLinkErrors(t *testing.T) {
	for _, scenario := range []string{"missing parent", "nonempty directory"} {
		t.Run(scenario, func(t *testing.T) {
			setupConfig(t)
			dir := t.TempDir()
			link := filepath.Join(dir, "missing", "current.log")
			wantError := "failed to create symlink"
			if scenario == "nonempty directory" {
				link = filepath.Join(dir, "existing")
				require.NoError(t, os.Mkdir(link, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(link, "keep"), []byte("keep"), 0600))
				wantError = "failed to remove existing symlink"
			}
			vip.Set(LogOutputTypeKey, []string{"file"})
			vip.Set(LogOutputCurrentLinkKey, link)
			writer, err := createWriterSyncer()
			require.ErrorContains(t, err, wantError)
			require.Nil(t, writer)
			if scenario == "nonempty directory" {
				data, err := os.ReadFile(filepath.Join(link, "keep"))
				require.NoError(t, err)
				require.Equal(t, "keep", string(data))
			}
		})
	}
}

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

func setupConfig(t *testing.T) {
	t.Helper()
	oldConfig, oldVip, oldLogger := config.Vip(), vip, zap.L()
	t.Cleanup(func() { config.SetVip(oldConfig); vip = oldVip; zap.ReplaceGlobals(oldLogger) })
	vip = viper.New()
	vip.SetEnvPrefix("SNET")
	vip.AutomaticEnv()

	defaults := viper.New()
	err := config.ReadConfigFromJsonString(defaults, `{"log":`+defaultLogConfigJSON+`}`)
	if err != nil {
		panic(fmt.Sprintf("Cannot load default config: %v", err))
	}
	config.SetDefaultFromConfig(vip, defaults)

	vip.AddConfigPath(".")

	config.SetVip(vip)
	vip.Set(LogOutputTypeKey, []string{"stdout"})
	vip.Set(LogOutputCurrentLinkKey, "")
	vip.Set(LogOutputFilePatternKey, filepath.Join(t.TempDir(), "daemon.%Y%m%d.log"))
}

type testGetLocationTimezone struct {
	name          string
	timezone      string
	expectedError string
}

func TestGetLocationTimezone(t *testing.T) {
	setupConfig(t)

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
	setupConfig(t)

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
			vip.Set(LogTimestampFormatKey, tc.timeStampFormat)

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
	setupConfig(t)

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
	setupConfig(t)

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
			vip.Set(LogOutputFilePatternKey, filepath.Join(t.TempDir(), filepath.Base(tc.filePatternName)))
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

func TestInitialize(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		value     any
		wantPanic string
	}{
		{"valid", LogLevelKey, "info", ""},
		{"invalid level", LogLevelKey, "INVALID", "failed to get logger level"},
		{"invalid timezone", LogTimezoneKey, "INVALID", "failed to create encoder config"},
		{"invalid formatter", LogFormatterTypeKey, "INVALID", "failed to get encoder"},
		{"invalid output", LogOutputTypeKey, []string{"INVALID"}, "failed to get logger writer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupConfig(t)
			vip.Set(tc.key, tc.value)
			if tc.wantPanic != "" {
				defer func() { r := recover(); assert.NotNil(t, r); assert.Contains(t, fmt.Sprint(r), tc.wantPanic) }()
				Initialize()
			} else {
				Initialize()
				assert.True(t, zap.L().Core().Enabled(zap.InfoLevel))
				assert.False(t, zap.L().Core().Enabled(zap.DebugLevel))
			}
		})
	}
}
