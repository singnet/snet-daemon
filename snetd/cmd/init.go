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
		log.Info("Writing default configuration")
		if err := config.WriteConfig(); err != nil {
			log.WithError(err).Error("Cannot write default configuration")
		}
	},
}
