package cmd

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write default configuration to file",
	Long:  "Use this command to create default configuration file. Then update the file with your settings.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var configFile = cmd.Flags().Lookup("config").Value.String()

		log.WithField("configFile", configFile).Info("Writing default configuration to the file")

		if isFileExist(configFile) {
			log.WithField("configFile", configFile).Fatal("configFile already exists, please remove file first")
		}

		err = config.WriteConfig(configFile)
		if err != nil {
			log.WithError(err).WithField("configFile", configFile).Fatal("Cannot write default configuration")
		}
	},
}
