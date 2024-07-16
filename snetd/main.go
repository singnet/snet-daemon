package main

import (
	_ "github.com/singnet/snet-daemon/fix-proto"
	"github.com/singnet/snet-daemon/snetd/cmd"
	"go.uber.org/zap"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		zap.L().Fatal("Unable to run application", zap.Error(err))
	}
}
