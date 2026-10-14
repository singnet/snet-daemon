package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func TestVerifyConfigCommandAcceptsValidConfiguration(t *testing.T) {
	prepareVerifyConfigTestConfiguration(t)
	configFile := writeVerifyConfigTestFile(t, `{}`)

	output, err := executeVerifyConfigCommand(t, configFile)

	require.NoError(t, err)
	var result configValidationOutput
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.True(t, result.Valid)
	require.Equal(t, configFile, result.Source)
	require.Zero(t, result.Summary.Errors)
	require.Empty(t, result.Diagnostics)
}

func TestVerifyConfigCommandReportsInvalidConfiguration(t *testing.T) {
	prepareVerifyConfigTestConfiguration(t)
	configFile := writeVerifyConfigTestFile(t, `{
  "daemon_type": "invalid",
  "max_message_size_in_mb": 0
}`)

	output, err := executeVerifyConfigCommand(t, configFile)

	require.ErrorIs(t, err, ErrInvalidConfiguration)
	var result configValidationOutput
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.False(t, result.Valid)
	require.Equal(t, 2, result.Summary.Errors)
	requireDiagnosticCode(t, result.Diagnostics, "invalid_daemon_type")
	requireDiagnosticCode(t, result.Diagnostics, "invalid_max_message_size")
}

func executeVerifyConfigCommand(t *testing.T, configFile string) (*bytes.Buffer, error) {
	t.Helper()
	root := &cobra.Command{Use: "snetd", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().StringP("config", "c", "snetd.config.json", "config file")
	root.AddCommand(newVerifyConfigCommand())
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetArgs([]string{"verify-config", "--config", configFile, "--format", "json"})
	return output, root.Execute()
}

func writeVerifyConfigTestFile(t *testing.T, contents string) string {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(configFile, []byte(contents), 0600))
	return configFile
}

func prepareVerifyConfigTestConfiguration(t *testing.T) {
	t.Helper()
	lifecycleConfig(t)
	v := viper.New()
	v.SetDefault(config.AllowedUserFlag, false)
	v.SetDefault(config.BlockchainEnabledKey, false)
	v.SetDefault(config.BlockChainNetworkSelected, "sepolia")
	v.SetDefault(config.DaemonEndpoint, "127.0.0.1:0")
	v.SetDefault(config.DaemonTypeKey, "grpc")
	v.SetDefault(config.MaxMessageSizeInMB, 4)
	v.SetDefault(config.MeteringEnabled, false)
	v.SetDefault(config.SSLCertPathKey, "")
	v.SetDefault(config.SSLKeyPathKey, "")
	v.SetDefault(config.ServiceEndpointKey, "http://127.0.0.1:1")
	v.SetDefault(config.ServiceHeartbeatType, "none")
	v.SetDefault(config.ServiceTimeout, "100s")
	config.SetVip(v)
}

func requireDiagnosticCode(t *testing.T, diagnostics []config.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic code %q not found in %+v", code, diagnostics)
}
