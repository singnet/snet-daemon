package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/singnet/snet-daemon/v6/snetd/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		if !errors.Is(err, cmd.ErrInvalidConfiguration) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}
