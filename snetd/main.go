package main

import (
	"os"

	"github.com/singnet/snet-daemon/snetd/cmd"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.WithError(err).Error("Unable to run application")
		os.Exit(1)
	}
}
