package main

import (
	"github.com/singnet/snet-daemon/snetd/cmd"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.WithError(err).Fatal("Unable to run application")
	}
}
