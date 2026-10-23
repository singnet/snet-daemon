package logger

import (
	"slices"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"
)

// ValidateConfiguration checks logger settings without opening outputs or
// replacing the process logger.
func ValidateConfiguration() config.ValidationReport {
	report := config.ValidationReport{Diagnostics: []config.Diagnostic{}}
	if _, err := getLoggerLevel(config.GetString(LogLevelKey)); err != nil {
		report.AddError("invalid_log_level", LogLevelKey, err.Error(), "Use debug, info, warn, warning, error, panic, or fatal.")
	}
	if _, err := getLocationTimezone(); err != nil {
		report.AddError("invalid_log_timezone", LogTimezoneKey, err.Error(), "Use UTC, Local, or an installed IANA time zone.")
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	if _, err := createEncoder(&encoderConfig); err != nil {
		report.AddError("invalid_log_formatter", LogFormatterTypeKey, err.Error(), "Use text or json.")
	}
	if _, err := configuredOutputTypes(); err != nil {
		report.AddError("invalid_log_output", LogOutputTypeKey, err.Error(), "Select one or more of stdout, stderr, and file.")
	}
	if slices.Contains(config.GetStringSlice(LogOutputTypeKey), "file") {
		if _, err := formatFileName(config.GetString(LogOutputFilePatternKey), time.Now()); err != nil {
			report.AddError("invalid_log_file_pattern", LogOutputFilePatternKey, err.Error(), "Use only %Y, %m, %d, %H, %M, and %S placeholders.")
		}
	}
	return report
}
