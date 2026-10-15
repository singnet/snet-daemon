package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidateConfigurationCollectsLoggerErrors(t *testing.T) {
	setupConfig(t)
	vip.Set(LogLevelKey, "invalid")
	vip.Set(LogTimezoneKey, "Invalid/Timezone")
	vip.Set(LogFormatterTypeKey, "invalid")
	vip.Set(LogOutputTypeKey, []string{"file", "invalid"})
	vip.Set(LogOutputFilePatternKey, "daemon.%Q.log")

	report := ValidateConfiguration()

	require.Equal(t, 5, report.Count(config.DiagnosticError))
	var fields []string
	for _, diagnostic := range report.Diagnostics {
		fields = append(fields, diagnostic.Field)
	}
	require.ElementsMatch(t, []string{LogLevelKey, LogTimezoneKey, LogFormatterTypeKey, LogOutputTypeKey, LogOutputFilePatternKey}, fields)
}

func TestValidateConfigurationRejectsEmptyOutputs(t *testing.T) {
	setupConfig(t)
	vip.Set(LogOutputTypeKey, []string{})
	report := ValidateConfiguration()
	require.Equal(t, 1, report.Count(config.DiagnosticError))
	require.Equal(t, LogOutputTypeKey, report.Diagnostics[0].Field)
}

func TestValidateConfigurationDoesNotInitializeLogger(t *testing.T) {
	setupConfig(t)
	dir := t.TempDir()
	logFile := filepath.Join(dir, "daemon.log")
	currentLink := filepath.Join(dir, "current.log")
	require.NoError(t, os.WriteFile(currentLink, []byte("preserve me"), 0600))
	vip.Set(LogOutputTypeKey, []string{"file", "stdout", "stderr"})
	vip.Set(LogOutputFilePatternKey, logFile)
	vip.Set(LogOutputCurrentLinkKey, currentLink)
	previousLogger := zap.L()

	for _, level := range []string{"debug", "info", "warn", "warning", "error", "panic", "fatal"} {
		vip.Set(LogLevelKey, level)
		require.Empty(t, ValidateConfiguration().Diagnostics)
	}
	vip.Set(LogFormatterTypeKey, "text")
	require.Empty(t, ValidateConfiguration().Diagnostics)
	require.Same(t, previousLogger, zap.L())
	require.NoFileExists(t, logFile)
	contents, err := os.ReadFile(currentLink)
	require.NoError(t, err)
	require.Equal(t, "preserve me", string(contents))
}
