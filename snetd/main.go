package main

import (
	"os"

	"github.com/singnet/snet-daemon/snetd/cmd"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.ServeCmd.Execute(); err != nil {
		log.WithError(err).Error("Unable to serve")
		os.Exit(1)
	}
}
