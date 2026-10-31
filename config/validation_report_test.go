package config

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfigurationCollectsIndependentDiagnostics(t *testing.T) {
	v := isolatedConfig(t)
	networkIdNameMapping = defaultBlockChainNetworkConfig
	stubVersionHTTP(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("static configuration validation must not check the latest release")
		return nil, nil
	})
	v.Set(DaemonTypeKey, "unsupported")
	v.Set(BlockChainNetworkSelected, "unknown")
	v.Set(SSLCertPathKey, "cert.pem")
	v.Set(MaxMessageSizeInMB, 0)
	v.Set(MeteringEnabled, true)
	v.Set(MeteringEndpoint, "invalid")
	v.Set(ServiceTimeout, "not-a-duration")

	report := ValidateConfiguration()

	require.True(t, report.HasErrors())
	require.Equal(t, 5, report.Count(DiagnosticError))
	require.Equal(t, 1, report.Count(DiagnosticWarning))
	requireDiagnosticCode(t, report, "invalid_daemon_type")
	requireDiagnosticCode(t, report, "invalid_blockchain_network")
	requireDiagnosticCode(t, report, "incomplete_ssl_configuration")
	requireDiagnosticCode(t, report, "invalid_max_message_size")
	requireDiagnosticCode(t, report, "invalid_metering_endpoint")
	requireDiagnosticCode(t, report, "invalid_duration")
}

func TestValidateConfigurationReportsWarningsWithoutErrors(t *testing.T) {
	v := isolatedConfig(t)
	networkIdNameMapping = defaultBlockChainNetworkConfig
	v.Set(DaemonTypeKey, "http")
	v.Set(ServiceTimeout, 5)

	report := ValidateConfiguration()

	require.False(t, report.HasErrors())
	require.Equal(t, 0, report.Count(DiagnosticError))
	require.Equal(t, 2, report.Count(DiagnosticWarning))
	requireDiagnosticCode(t, report, "http_daemon_type")
	requireDiagnosticCode(t, report, "invalid_duration_type")
}

func TestValidationReportCountsDiagnosticsAndReturnsFirstError(t *testing.T) {
	report := ValidationReport{}
	report.AddWarning("warning", "field", "warning message", "")
	report.AddError("first", "field", "first error", "")
	report.AddError("second", "field", "second error", "")

	require.True(t, report.HasErrors())
	require.Equal(t, 2, report.Count(DiagnosticError))
	require.Equal(t, 1, report.Count(DiagnosticWarning))
	require.EqualError(t, report.Error(), "first error")
}

func TestDetermineNetworkSelectedRejectsUnknownNetwork(t *testing.T) {
	v := isolatedConfig(t)
	v.Set(BlockChainNetworkSelected, "unknown")

	require.ErrorContains(t, determineNetworkSelected([]byte(defaultBlockChainNetworkConfig)), "not defined")
}

func requireDiagnosticCode(t *testing.T, report ValidationReport, code string) {
	t.Helper()
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic code %q not found in %+v", code, report.Diagnostics)
}
