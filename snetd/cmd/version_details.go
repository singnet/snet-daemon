package cmd

import (
	"fmt"
	"io"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
)

// VersionCmd prints build and revision metadata for the daemon.
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "List the current version of the Daemon.",
	Long:  "To check the current version of the Daemon, the sha1 revision and the time the Binary was built User can use `snetd version`",
	RunE: func(cmd *cobra.Command, args []string) error {
		command, err := newListVersionCommand(cmd, args, nil)
		if err != nil {
			return err
		}
		return command.Run()
	},
}

type ListVersionCommand struct {
	out io.Writer
}

func newListVersionCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	command = &ListVersionCommand{out: cmd.OutOrStdout()}
	return
}

func (command *ListVersionCommand) Run() (err error) {
	_, err = fmt.Fprintf(command.out, "version tag: %s\nbuilt on: %s\nsha1 revision: %s\n",
		config.GetVersionTag(), config.GetBuildTime(), config.GetSha1Revision())
	return err
}
