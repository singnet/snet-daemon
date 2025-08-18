package logger

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LogLevelKey    = "log.level"
	LogTimezoneKey = "log.timezone"

	LogFormatterTypeKey   = "log.formatter.type"
	LogTimestampFormatKey = "log.formatter.timestamp_format"

	LogOutputTypeKey        = "log.output.type"
	LogOutputFilePatternKey = "log.output.file_pattern"
	LogOutputCurrentLinkKey = "log.output.current_link"
	LogMaxSizeKey           = "log.output.max_size_in_mb"
	LogMaxAgeKey            = "log.output.max_age_in_days"
	LogRotationCountKey     = "log.output.rotation_count"
)

// InitLogger initializes logger using configuration provided by viper
// instance.
//
// Function designed to configure a few different loggers with different
// formatter and output settings. To achieve, this viper configuration
// contains separate sections for each logger, each output and
// each formatter.

func Initialize() {

	levelString := config.GetString(LogLevelKey)
	level, err := getLoggerLevel(levelString)
	if err != nil {
		panic(fmt.Errorf("failed to get logger level: %v", err))
	}

	encoderConfig, err := createEncoderConfig()
	if err != nil {
		panic(fmt.Errorf("failed to create encoder config, error: %v", err))
	}

	encoder, err := createEncoder(encoderConfig)
	if err != nil {
		panic(fmt.Errorf("failed to get encoder, error: %v", err))
	}

	writerSyncer, err := createWriterSyncer()
	if err != nil {
		panic(fmt.Errorf("failed to get logger writer, error: %v", err))
	}

	core := zapcore.NewCore(encoder, writerSyncer, level)
	logger := zap.New(core)

	var hooks []zap.Option

	for _, hookConfigName := range config.GetStringSlice(LogHooksKey) {
		hook, err := initHookByConfig(config.Vip().Sub(config.LogKey + "." + hookConfigName))
		if err != nil {
			fmt.Printf("unable to add log hook \"%v\", error: %v", hookConfigName, err)
		}
		hooks = append(hooks, hook)
	}

	zap.ReplaceGlobals(logger.WithOptions(hooks...))

	zap.L().Info("Logger initialized", zap.String("level", levelString))
}

func getLoggerLevel(levelString string) (zapcore.Level, error) {
	switch levelString {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn":
		return zap.WarnLevel, nil
	case "warning":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	case "fatal":
		return zap.FatalLevel, nil
	case "panic":
		return zap.PanicLevel, nil
	default:
		return zapcore.Level(0), fmt.Errorf("wrong string for level: %v. Available options: debug, info, warn, error, panic", levelString)
	}
}

func createEncoderConfig() (*zapcore.EncoderConfig, error) {
	location, err := getLocationTimezone()
	if err != nil {
		return nil, err
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	if config.GetString(LogTimestampFormatKey) != "" {
		customTimeFormat := config.GetString(LogTimestampFormatKey)
		encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.In(location).Format(customTimeFormat))
		}
	} else {
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
		encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.In(location).Format(time.RFC3339))
		}
	}
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// colorless logs for files
	if slices.Contains(config.GetStringSlice(LogOutputTypeKey), "file") {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder // "12.345ms"

	return &encoderConfig, nil
}

func getLocationTimezone() (location *time.Location, err error) {
	timezone := config.GetString(LogTimezoneKey)
	location, err = time.LoadLocation(timezone)
	return
}

func formatFileName(pattern string, now time.Time) (string, error) {
	formatMap := map[string]string{
		"%Y": fmt.Sprintf("%04d", now.Year()),
		"%m": fmt.Sprintf("%02d", int(now.Month())),
		"%d": fmt.Sprintf("%02d", now.Day()),
		"%H": fmt.Sprintf("%02d", now.Hour()),
		"%M": fmt.Sprintf("%02d", now.Minute()),
		"%S": fmt.Sprintf("%02d", now.Second()),
	}

	parts := strings.Split(pattern, "%")

	for _, part := range parts[1:] {
		if len(part) > 0 && !strings.ContainsAny(part[0:1], "YmdHMS") {
			return "", errors.New("invalid placeholder found in pattern: %" + part[0:1])
		}
	}

	for placeholder, value := range formatMap {
		pattern = strings.ReplaceAll(pattern, placeholder, value)
	}

	return pattern, nil
}

func createEncoder(encoderConfig *zapcore.EncoderConfig) (zapcore.Encoder, error) {
	var encoder zapcore.Encoder
	switch config.GetString(LogFormatterTypeKey) {
	case "json":
		encoder = zapcore.NewJSONEncoder(*encoderConfig)
	case "text":
		encoder = zapcore.NewConsoleEncoder(*encoderConfig)
	default:
		return nil, fmt.Errorf("unsupported log formatter type: %v", config.GetString(LogFormatterTypeKey))
	}
	return encoder, nil
}

func createWriterSyncer() (zapcore.WriteSyncer, error) {
	var writers []zapcore.WriteSyncer

	configWriters := config.GetStringSlice(LogOutputTypeKey)

	if len(configWriters) == 0 {
		return nil, fmt.Errorf("failed to read log.output.type from config: %v", configWriters)
	}

	for _, writer := range configWriters {
		switch writer {
		case "stdout":
			writers = append(writers, zapcore.AddSync(os.Stdout))
		case "stderr":
			writers = append(writers, zapcore.AddSync(os.Stderr))
		case "file":
			fileName, err := formatFileName(config.GetString(LogOutputFilePatternKey), time.Now())
			if err != nil {
				return nil, fmt.Errorf("failed to create file writer for logger, %v", err)
			}
			writers = append(writers, zapcore.AddSync(&lumberjack.Logger{
				Filename:   fileName,
				MaxSize:    config.GetInt(LogMaxSizeKey),
				MaxAge:     config.GetInt(LogMaxAgeKey),
				MaxBackups: config.GetInt(LogRotationCountKey),
				Compress:   true,
			}))

			currentLink := config.GetString(LogOutputCurrentLinkKey)
			if currentLink != "" {
				if _, err := os.Lstat(currentLink); err == nil {
					if err := os.Remove(currentLink); err != nil {
						return nil, fmt.Errorf("failed to remove existing symlink: %v", err)
					}
				}
				if err := os.Symlink(fileName, currentLink); err != nil {
					return nil, fmt.Errorf("failed to create symlink: %v", err)
				}
			}
		default:
			return nil, fmt.Errorf("unsupported log output type: %v", writer)
		}
	}

	multipleWs := zapcore.NewMultiWriteSyncer(writers...)

	return multipleWs, nil
}
