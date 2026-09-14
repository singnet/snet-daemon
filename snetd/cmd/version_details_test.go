package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingVersionWriter struct{}

func (failingVersionWriter) Write([]byte) (int, error) {
	return 0, assert.AnError
}

func TestNewListVersionCommand(t *testing.T) {
	var output bytes.Buffer
	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(&output)

	command, err := newListVersionCommand(cobraCmd, nil, nil)
	require.NoError(t, err)

	versionCommand, ok := command.(*ListVersionCommand)
	require.True(t, ok)
	assert.Same(t, &output, versionCommand.out)
}

func TestVersionCmdRunE(t *testing.T) {
	var output bytes.Buffer
	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(&output)

	err := VersionCmd.RunE(cobraCmd, nil)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("version tag: %s\nbuilt on: %s\nsha1 revision: %s\n",
		config.GetVersionTag(), config.GetBuildTime(), config.GetSha1Revision()), output.String())
}

func TestListVersionCommandRunReturnsWriteError(t *testing.T) {
	command := &ListVersionCommand{out: failingVersionWriter{}}

	assert.ErrorIs(t, command.Run(), assert.AnError)
}
