package main

import (
	"errors"
	"os"

	"github.com/singnet/snet-daemon/v6/snetd/cmd"
	"go.uber.org/zap"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		if errors.Is(err, cmd.ErrInvalidConfiguration) {
			os.Exit(1)
		}
		zap.L().Fatal("Unable to run application", zap.Error(err))
	}
}
