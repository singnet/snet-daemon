package cmd

import (
	"os"

	"github.com/singnet/snet-daemon/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write default configuration to file",
	Long:  "Use this command to create simple configuration file. Then update the file with your settings.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var configFile = cmd.Flags().Lookup("config").Value.String()

		zap.L().Info("Writing default configuration to the file", zap.String("config", configFile))

		if isFileExist(configFile) {
			zap.L().Fatal("such configFile already exists, please remove file first or rename file",
				zap.String("configFile", configFile))
		}

		err = os.WriteFile(configFile, []byte(config.MinimumConfigJson), 0666)
		if err != nil {
			zap.L().Fatal("Cannot write configuration", zap.String("config", configFile), zap.Error(err))
		}
	},
}

var InitFullCmd = &cobra.Command{
	Use:   "init-full",
	Short: "Write full default configuration to file",
	Long:  "Use this command to create full default configuration file. Then update the file with your settings.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var configFile = cmd.Flags().Lookup("config").Value.String()

		zap.L().Info("Writing full configuration to the file", zap.String("config", configFile))

		if isFileExist(configFile) {
			zap.L().Fatal("such configFile already exists, please remove file first or rename file", zap.String("config", configFile))
		}

		err = config.WriteConfig(configFile)
		if err != nil {
			zap.L().Fatal("Cannot write full configuration", zap.Error(err), zap.String("config", configFile))
		}
	},
}
