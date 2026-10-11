package cmd

import (
	"fmt"
	"os"

	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write default configuration to file",
	Long:  "Use this command to create simple configuration file. Then update the file with your settings.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := configFileFromCmd(cmd)

		if isFileExist(configFile) {
			return fmt.Errorf("ERROR: such configFile already exists, please remove file first or rename file: %s", configFile)
		}

		if err := os.WriteFile(configFile, []byte(config.MinimumConfigJson), 0666); err != nil {
			return fmt.Errorf("ERROR: Cannot write configuration rename file: %s", configFile)
		}
		fmt.Println("Writing basic configuration to the file:", configFile)
		return nil
	},
}

var InitFullCmd = &cobra.Command{
	Use:   "init-full",
	Short: "Write full default configuration to file",
	Long:  "Use this command to create full default configuration file. Then update the file with your settings.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := configFileFromCmd(cmd)

		if isFileExist(configFile) {
			return fmt.Errorf("ERROR: such configFile already exists, please remove file first or rename file: %s", configFile)
		}

		if err := config.WriteConfig(configFile); err != nil {
			return fmt.Errorf("ERROR: Cannot write full configuration to file: %s", configFile)
		}

		fmt.Println("Writing full default configuration to the file:", configFile)
		return nil
	},
}

func configFileFromCmd(cmd *cobra.Command) string {
	return cmd.Flags().Lookup("config").Value.String()
}
