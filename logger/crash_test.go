package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	loggerProcessEnv = "SNET_LOGGER_PROCESS_TEST"
	crashMessage     = "daemon process test failure"
	crashCause       = "process test root cause"
	crashRequest     = "process-test-request"
)

type loggerProcessConfig struct {
	Action    string
	Level     string
	Formatter string
	Outputs   []string
	LogFile   string
	HookFile  string
}

type processRecordingHook struct{ file *os.File }

func (h processRecordingHook) call(entry zapcore.Entry) error {
	_, err := fmt.Fprintln(h.file, entry.Level.String())
	return err
}

// Run before the test runner so it cannot recover the panic or add output of its own.
func init() {
	encoded := os.Getenv(loggerProcessEnv)
	if encoded == "" {
		return
	}
	var settings loggerProcessConfig
	if err := json.Unmarshal([]byte(encoded), &settings); err != nil {
		panic(err)
	}
	conf := viper.New()
	conf.Set(LogLevelKey, settings.Level)
	conf.Set(LogTimezoneKey, "UTC")
	conf.Set(LogFormatterTypeKey, settings.Formatter)
	conf.Set(LogOutputTypeKey, settings.Outputs)
	conf.Set(LogOutputFilePatternKey, settings.LogFile)
	conf.Set(LogOutputCurrentLinkKey, "")
	config.SetVip(conf)

	hookFile, err := os.OpenFile(settings.HookFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer func() { _ = hookFile.Close() }()
	const hookType = "process-recording-test"
	RegisterHookType(hookType, func(*viper.Viper) (hook, error) {
		return processRecordingHook{file: hookFile}, nil
	})
	conf.Set(LogHooksKey, []string{"capture"})
	conf.Set("log.capture", map[string]any{
		"type": hookType, "levels": []string{"panic", "fatal"},
	})
	Initialize()
	log := zap.L().With(zap.String("request", crashRequest))
	switch settings.Action {
	case "panic":
		log.Panic(crashMessage, zap.Error(errors.New(crashCause)))
	case "fatal":
		log.Fatal(crashMessage, zap.Error(errors.New(crashCause)))
	case "error":
		log.Error(crashMessage, zap.Error(errors.New(crashCause)))
	default:
		panic(fmt.Sprintf("unknown process action: %q", settings.Action))
	}
	os.Exit(0)
}

func runLoggerProcess(t *testing.T, settings loggerProcessConfig) (string, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	settings.LogFile = filepath.Join(dir, "daemon.log")
	settings.HookFile = filepath.Join(dir, "hooks.log")
	encoded, err := json.Marshal(settings)
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^$")
	command.Env = append(os.Environ(), loggerProcessEnv+"="+string(encoded), "GOTRACEBACK=single")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	require.NoError(t, ctx.Err(), "logger helper timed out: %s", stderr.String())
	if settings.Action == "error" {
		require.NoError(t, err, "%s", stderr.String())
	} else {
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, "%s", stderr.String())
		if settings.Action == "fatal" {
			require.Equal(t, 1, exitErr.ExitCode())
		} else {
			require.NotZero(t, exitErr.ExitCode())
			require.Contains(t, stderr.String(), "panic: "+crashMessage)
			require.Contains(t, stderr.String(), "goroutine ")
		}
	}
	var fileLog string
	if slices.Contains(settings.Outputs, "file") {
		data, err := os.ReadFile(settings.LogFile)
		require.NoError(t, err)
		fileLog = string(data)
	} else {
		require.NoFileExists(t, settings.LogFile)
	}
	hookLog, err := os.ReadFile(settings.HookFile)
	require.NoError(t, err)
	return stdout.String(), stderr.String(), fileLog, string(hookLog)
}

// Count encoded records separately from the runtime's panic message and traceback.
func assertProcessLog(t *testing.T, output, formatter, level string, count int) {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(output, "\n") {
		var fields map[string]any
		if formatter == "json" {
			if !strings.HasPrefix(line, "{") {
				continue
			}
			require.NoError(t, json.Unmarshal([]byte(line), &fields))
			if fields["msg"] != crashMessage {
				continue
			}
		} else {
			parts := strings.SplitN(line, "\t", 4)
			if len(parts) != 4 || parts[2] != crashMessage {
				continue
			}
			require.NoError(t, json.Unmarshal([]byte(parts[3]), &fields))
			fields["level"] = parts[1]
		}
		require.Contains(t, fields["level"], strings.ToUpper(level))
		require.Equal(t, crashCause, fields["error"])
		require.Equal(t, crashRequest, fields["request"])
		require.Equal(t, 1, strings.Count(line, `"request"`))
		records = append(records, fields)
	}
	require.Len(t, records, count, "unexpected log records in:\n%s", output)
}

func TestLoggerCrashOutputs(t *testing.T) {
	for _, formatter := range []string{"json", "text"} {
		for _, level := range []string{"info", "fatal"} {
			for _, outputs := range [][]string{
				{"file"}, {"stdout"}, {"stderr"}, {"file", "stdout"}, {"file", "stdout", "stderr"},
			} {
				for _, action := range []string{"panic", "fatal"} {
					name := strings.Join([]string{formatter, level, strings.Join(outputs, "+"), action}, "/")
					t.Run(name, func(t *testing.T) {
						stdout, stderr, fileLog, hookLog := runLoggerProcess(t, loggerProcessConfig{
							Action: action, Level: level, Formatter: formatter, Outputs: outputs,
						})
						assertProcessLog(t, stderr, formatter, action, 1)
						if slices.Contains(outputs, "stdout") {
							assertProcessLog(t, stdout, formatter, action, 1)
						} else {
							require.Empty(t, stdout)
						}
						if slices.Contains(outputs, "file") {
							assertProcessLog(t, fileLog, formatter, action, 1)
						}
						require.Equal(t, action+"\n", hookLog)
					})
				}
			}
		}
	}
}

func TestLoggerErrorRemainsInConfiguredFile(t *testing.T) {
	for _, formatter := range []string{"json", "text"} {
		t.Run(formatter, func(t *testing.T) {
			stdout, stderr, fileLog, hookLog := runLoggerProcess(t, loggerProcessConfig{
				Action: "error", Level: "info", Formatter: formatter, Outputs: []string{"file"},
			})
			require.Empty(t, stdout)
			require.Empty(t, stderr)
			assertProcessLog(t, fileLog, formatter, "error", 1)
			require.Empty(t, hookLog)
		})
	}
}

func TestLoggerFatalLevelFiltersOrdinaryRecords(t *testing.T) {
	setupConfig(t)
	fileName := filepath.Join(t.TempDir(), "filtered.log")
	vip.Set(LogLevelKey, "fatal")
	vip.Set(LogOutputTypeKey, []string{"file"})
	vip.Set(LogOutputFilePatternKey, fileName)
	Initialize()
	for _, level := range []zapcore.Level{
		zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel, zapcore.DPanicLevel,
	} {
		zap.L().Log(level, "filtered ordinary record")
	}
	require.NoFileExists(t, fileName)
}
