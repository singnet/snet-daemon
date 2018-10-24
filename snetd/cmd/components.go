package cmd

import (
	"github.com/singnet/snet-daemon/escrow"
	"github.com/spf13/cobra"
)

type Components struct {
}

func InitComponents(cmd *cobra.Command) *Components {
	// TODO: implement
	return nil
}

func (components *Components) Close() {
}

func (components *Components) PaymentChannelStorage() (storage escrow.PaymentChannelStorage) {
	// TODO: implement
	return nil
}
