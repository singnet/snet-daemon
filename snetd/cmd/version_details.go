package cmd

import (
	"fmt"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
	"os"
)

// Shows the current version of the Daemons
// Version tag: v0.1.4-181-g2fd9f04 was build on: 2019-03-19_13:52:58 with sha1 revision from github: 2fd9f04bfb279aaf66291cd6bd2ca734fd4f70b5
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "List the current version of the Daemon.",
	Long:  "To check the current version of the Daemon, the sha1 revision and the time the Binary was built User can use `snetd version`",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newListVersionCommand)
	},
}

type ListVersionCommand struct {
}

func newListVersionCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	command = &ListVersionCommand{}
	return
}

func (command *ListVersionCommand) Run() (err error) {
	fmt.Printf("version tag: %s\n", config.GetVersionTag())
	fmt.Printf("built on: %s\n", config.GetBuildTime())
	fmt.Printf("sha1 revision: %s\n", config.GetSha1Revision())
	os.Exit(0)
	return nil
}
