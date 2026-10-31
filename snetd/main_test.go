package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/snetd/cmd"
	"github.com/stretchr/testify/require"
)

const cliProcessEnv = "SNET_CLI_PROCESS_TEST"

// Execute the real entry point before the test runner can change its output.
func init() {
	encoded := os.Getenv(cliProcessEnv)
	if encoded == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(encoded), &args); err != nil {
		panic(err)
	}
	cmd.RootCmd.SetArgs(args)
	main()
	os.Exit(0)
}

func runCLIProcess(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	repoRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^$")
	command.Dir = repoRoot
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(entry), "SNET_") {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, cliProcessEnv+"="+string(encoded))
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		require.ErrorAs(t, err, &exitError, "%s", stderr.String())
	}
	require.NoError(t, ctx.Err(), "CLI helper timed out: %s", stderr.String())
	return command.ProcessState.ExitCode(), stdout.String(), stderr.String()
}

func TestVerifyConfigCLI(t *testing.T) {
	for _, tc := range []struct {
		name      string
		contents  string
		args      []string
		wantError string
		wantCodes []string
	}{
		{name: "valid JSON", contents: `{}`, args: []string{"--format", "json"}},
		{name: "invalid port", contents: `{"daemon_endpoint":"127.0.0.1:70000"}`, args: []string{"--format", "json"}, wantCodes: []string{"invalid_service_endpoint"}},
		{name: "invalid log level", contents: `{"log":{"level":"invalid"}}`, args: []string{"--format", "json"}, wantCodes: []string{"invalid_log_level"}},
		{name: "unreadable configuration", args: []string{"--format", "json"}, wantCodes: []string{"config_load_failed"}},
		{name: "malformed configuration", contents: `{`, args: []string{"--format", "json"}, wantCodes: []string{"config_load_failed"}},
		{name: "invalid format", contents: `{}`, args: []string{"--format", "yaml"}, wantError: `unsupported format "yaml"; use text or json`},
		{name: "unknown flag", contents: `{}`, args: []string{"--unknown"}, wantError: "unknown flag: --unknown"},
		{name: "unexpected argument", contents: `{}`, args: []string{"extra"}, wantError: `unknown command "extra"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config.json")
			if tc.contents != "" {
				require.NoError(t, os.WriteFile(configFile, []byte(tc.contents), 0600))
			}
			args := append([]string{"verify-config", "--config", configFile}, tc.args...)
			exitCode, stdout, stderr := runCLIProcess(t, args...)
			if tc.wantError != "" {
				require.Equal(t, 1, exitCode)
				require.Empty(t, stdout)
				require.Equal(t, 1, strings.Count(stderr, tc.wantError), "%s", stderr)
				require.NotContains(t, stderr, "Usage:")
				return
			}
			require.Empty(t, stderr)
			var report struct {
				Valid       bool                `json:"valid"`
				Diagnostics []config.Diagnostic `json:"diagnostics"`
			}
			require.NoError(t, json.Unmarshal([]byte(stdout), &report))
			require.Equal(t, len(tc.wantCodes) == 0, report.Valid)
			if report.Valid {
				require.Zero(t, exitCode)
			} else {
				require.Equal(t, 1, exitCode)
			}
			var codes []string
			for _, diagnostic := range report.Diagnostics {
				codes = append(codes, diagnostic.Code)
			}
			require.ElementsMatch(t, tc.wantCodes, codes)
		})
	}
}

func TestServeCLIReportsAllConfigurationErrorsBeforeLoggerInitialization(t *testing.T) {
	for _, command := range []string{"serve", ""} {
		t.Run("command="+command, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, os.WriteFile(configFile, []byte(`{
				"daemon_type":"invalid",
				"daemon_endpoint":"127.0.0.1:70000",
				"max_message_size_in_mb":0,
				"log":{"level":"invalid"}
			}`), 0600))
			args := []string{"--config", configFile}
			if command != "" {
				args = append([]string{command}, args...)
			}
			exitCode, _, stderr := runCLIProcess(t, args...)
			require.Equal(t, 1, exitCode)
			require.Contains(t, stderr, "Result: INVALID - 4 errors")
			for _, code := range []string{"invalid_daemon_type", "invalid_service_endpoint", "invalid_max_message_size", "invalid_log_level"} {
				require.Equal(t, 1, strings.Count(stderr, code), "%s", stderr)
			}
			require.NotContains(t, stderr, "panic:")
			require.NotContains(t, stderr, "Usage:")
			require.NotContains(t, stderr, "Error: configuration is invalid")
		})
	}
}
