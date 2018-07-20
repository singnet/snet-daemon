package main

import (
	"os"

	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/snetd/cmd"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.Level(config.GetInt(config.LogLevelKey)))

	if err := cmd.ServeCmd.Execute(); err != nil {
		log.WithError(err).Error("Unable to serve")
		os.Exit(1)
	}
}
