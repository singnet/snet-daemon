package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/stretchr/testify/require"
)

func TestWriteConfigValidationReportText(t *testing.T) {
	var output bytes.Buffer
	report := config.ValidationReport{}

	require.NoError(t, writeConfigValidationReport(&output, "text", "config.json", report))
	require.Equal(t, "✅ Configuration: config.json\nResult: VALID\n", output.String())
}

func TestWriteConfigValidationReportTextWithError(t *testing.T) {
	var output bytes.Buffer
	report := config.ValidationReport{}
	report.AddError("invalid_endpoint", "daemon_endpoint", "invalid endpoint", "Use host:port.")

	require.NoError(t, writeConfigValidationReport(&output, "text", "config.json", report))
	require.Equal(t, "Configuration: config.json\nResult: INVALID - 1 errors, 0 warnings\n\nerror [daemon_endpoint] invalid_endpoint\n  invalid endpoint\n  Hint: Use host:port.\n", output.String())
}

func TestWriteConfigValidationReportJSON(t *testing.T) {
	var output bytes.Buffer
	report := config.ValidationReport{}
	report.AddWarning("deprecated_config_key", "old_key", "deprecated", "Use new_key.")

	require.NoError(t, writeConfigValidationReport(&output, "json", "config.json", report))

	var result configValidationOutput
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.True(t, result.Valid)
	require.Equal(t, "config.json", result.Source)
	require.Equal(t, 0, result.Summary.Errors)
	require.Equal(t, 1, result.Summary.Warnings)
	require.Len(t, result.Diagnostics, 1)
	require.Equal(t, "deprecated_config_key", result.Diagnostics[0].Code)
}
