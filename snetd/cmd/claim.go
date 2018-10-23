package cmd

import (
	"github.com/spf13/cobra"
)

var ClaimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim money from payment channel",
	Long:  "Increment payment channel nonce and send blockchain transaction to claim money from channel",
	Run: func(cmd *cobra.Command, args []string) {

	},
}
