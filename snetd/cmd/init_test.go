package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ensureConfigFlag(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	if cmd.Flags().Lookup("config") == nil {
		cmd.Flags().String("config", "", "config file")
	}
}

func TestInitCmdWritesMinimumConfig(t *testing.T) {
	ensureConfigFlag(t, InitCmd)
	cfgPath := filepath.Join(t.TempDir(), "snetd.config.json")
	require.NoError(t, InitCmd.Flags().Set("config", cfgPath))

	err := InitCmd.RunE(InitCmd, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, config.MinimumConfigJson, string(data))
}

func TestInitCmdFailsWhenConfigExists(t *testing.T) {
	ensureConfigFlag(t, InitCmd)
	cfgPath := filepath.Join(t.TempDir(), "snetd.config.json")
	require.NoError(t, os.WriteFile(cfgPath, []byte("{}"), 0644))
	require.NoError(t, InitCmd.Flags().Set("config", cfgPath))

	err := InitCmd.RunE(InitCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "such configFile already exists")
}

func TestInitCmdWriteFailure(t *testing.T) {
	ensureConfigFlag(t, InitCmd)
	cfgPath := filepath.Join(t.TempDir(), "missing-subdir", "snetd.config.json")
	require.NoError(t, InitCmd.Flags().Set("config", cfgPath))

	err := InitCmd.RunE(InitCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot write configuration")
}

func TestInitFullCmdWritesConfig(t *testing.T) {
	ensureConfigFlag(t, InitFullCmd)
	cfgPath := filepath.Join(t.TempDir(), "snetd.config.json")
	require.NoError(t, InitFullCmd.Flags().Set("config", cfgPath))

	err := InitFullCmd.RunE(InitFullCmd, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.NotEmpty(t, string(data))
}

func TestInitFullCmdFailsWhenConfigExists(t *testing.T) {
	ensureConfigFlag(t, InitFullCmd)
	cfgPath := filepath.Join(t.TempDir(), "snetd.config.json")
	require.NoError(t, os.WriteFile(cfgPath, []byte("{}"), 0644))
	require.NoError(t, InitFullCmd.Flags().Set("config", cfgPath))

	err := InitFullCmd.RunE(InitFullCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "such configFile already exists")
}

func TestInitFullCmdWriteFailure(t *testing.T) {
	ensureConfigFlag(t, InitFullCmd)
	cfgPath := filepath.Join(t.TempDir(), "missing-subdir", "snetd.config.json")
	require.NoError(t, InitFullCmd.Flags().Set("config", cfgPath))

	err := InitFullCmd.RunE(InitFullCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot write full configuration")
}
