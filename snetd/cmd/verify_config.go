package cmd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/metrics"
	"github.com/spf13/cobra"
)

var ErrInvalidConfiguration = errors.New("configuration is invalid")

var VerifyConfigCmd = newVerifyConfigCommand()

func newVerifyConfigCommand() *cobra.Command {
	var format string
	command := &cobra.Command{
		Use:           "verify-config",
		Short:         "Verify daemon configuration without starting the daemon",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("unsupported format %q; use text or json", format)
			}

			configFlag := cmd.Flags().Lookup("config")
			if configFlag == nil {
				configFlag = cmd.InheritedFlags().Lookup("config")
			}
			source, _, err := loadConfigFileFromCommandLine(configFlag)
			report := config.ValidationReport{Diagnostics: []config.Diagnostic{}}
			if err != nil {
				report.AddError("config_load_failed", "config", fmt.Sprintf("cannot read configuration file: %v", err), "Provide a readable configuration file in JSON, YAML, or TOML format.")
			} else {
				report = validateDaemonConfiguration()
			}

			if err := writeConfigValidationReport(cmd.OutOrStdout(), format, source, report); err != nil {
				return err
			}
			if report.HasErrors() {
				return ErrInvalidConfiguration
			}
			return nil
		},
	}
	command.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return command
}

func validateDaemonConfiguration() config.ValidationReport {
	report := config.ValidateConfiguration()
	validateTLSKeyPair(&report)

	if err := metrics.ValidateHeartbeatConfig(config.GetString(config.ServiceHeartbeatType), config.GetString(config.HeartbeatServiceEndpoint)); err != nil {
		report.AddError("invalid_heartbeat_configuration", config.ServiceHeartbeatType, err.Error(), "Use grpc, http, https, tcp, none, or leave the heartbeat configuration empty.")
	}
	if err := metrics.ValidateNotificationConfig(); err != nil {
		report.AddError("invalid_notification_configuration", config.NotificationServiceEndpoint+", "+config.AlertsEMail, err.Error(), "Set a valid notification_endpoint and alerts_email, or leave both empty.")
	}

	return report
}

func validateTLSKeyPair(report *config.ValidationReport) {
	certPath := config.GetString(config.SSLCertPathKey)
	keyPath := config.GetString(config.SSLKeyPathKey)
	if certPath == "" || keyPath == "" {
		return
	}
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		report.AddError("invalid_ssl_keypair", config.SSLCertPathKey+", "+config.SSLKeyPathKey, "unable to load SSL X509 key pair: "+err.Error(), "Ensure ssl_cert and ssl_key point to a matching, readable certificate and private key.")
	}
}

type configValidationOutput struct {
	Valid       bool                    `json:"valid"`
	Source      string                  `json:"source"`
	Summary     configValidationSummary `json:"summary"`
	Diagnostics []config.Diagnostic     `json:"diagnostics"`
}

type configValidationSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

func writeConfigValidationReport(writer io.Writer, format, source string, report config.ValidationReport) error {
	if format == "json" {
		return json.NewEncoder(writer).Encode(configValidationOutput{
			Valid:  !report.HasErrors(),
			Source: source,
			Summary: configValidationSummary{
				Errors:   report.Count(config.DiagnosticError),
				Warnings: report.Count(config.DiagnosticWarning),
			},
			Diagnostics: report.Diagnostics,
		})
	}

	errorsCount := report.Count(config.DiagnosticError)
	warningsCount := report.Count(config.DiagnosticWarning)
	if errorsCount == 0 {
		fmt.Fprint(writer, "✅ ")
	}
	fmt.Fprintf(writer, "Configuration: %s\n", source)
	switch {
	case errorsCount > 0:
		fmt.Fprintf(writer, "Result: INVALID - %d errors, %d warnings\n", errorsCount, warningsCount)
	case warningsCount > 0:
		fmt.Fprintf(writer, "Result: VALID WITH WARNINGS - %d warnings\n", warningsCount)
	default:
		fmt.Fprintln(writer, "Result: VALID")
	}

	for _, severity := range []config.DiagnosticSeverity{config.DiagnosticError, config.DiagnosticWarning} {
		for _, diagnostic := range report.Diagnostics {
			if diagnostic.Severity != severity {
				continue
			}
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, "%s [%s] %s\n", diagnostic.Severity, diagnostic.Field, diagnostic.Code)
			fmt.Fprintf(writer, "  %s\n", diagnostic.Message)
			if diagnostic.Hint != "" {
				fmt.Fprintf(writer, "  Hint: %s\n", diagnostic.Hint)
			}
		}
	}

	return nil
}
