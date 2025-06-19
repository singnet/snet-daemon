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
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var configFile = cmd.Flags().Lookup("config").Value.String()

		if isFileExist(configFile) {
			fmt.Println("ERROR: such configFile already exists, please remove file first or rename file:", configFile)
			os.Exit(-1)
		}

		err = os.WriteFile(configFile, []byte(config.MinimumConfigJson), 0666)
		if err != nil {
			fmt.Println("ERROR: Cannot write configuration rename file:", configFile)
			os.Exit(-1)
		}
		fmt.Println("Writing basic configuration to the file:", configFile)
	},
}

var InitFullCmd = &cobra.Command{
	Use:   "init-full",
	Short: "Write full default configuration to file",
	Long:  "Use this command to create full default configuration file. Then update the file with your settings.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var configFile = cmd.Flags().Lookup("config").Value.String()

		if isFileExist(configFile) {
			fmt.Println("ERROR: such configFile already exists, please remove file first or rename file:", configFile)
			os.Exit(-1)
		}

		err = config.WriteConfig(configFile)
		if err != nil {
			fmt.Println("ERROR: Cannot write full configuration to file:", configFile)
			os.Exit(-1)
		}

		fmt.Println("Writing full default configuration to the file:", configFile)
	},
}
