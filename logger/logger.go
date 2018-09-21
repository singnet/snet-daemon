package logger

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
)

func InitLogger() {
	var err error
	var logLevel log.Level
	var settings = config.GetString(config.LogLevelKey)

	logLevel, err = log.ParseLevel(settings)
	if err != nil {
		log.WithError(err).WithField("settings", settings).Fatal("Cannot parse log level value from settings")
	}

	log.SetLevel(logLevel)
}
