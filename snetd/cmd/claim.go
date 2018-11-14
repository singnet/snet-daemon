package cmd

import (
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/cobra"

	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/escrow"
)

var ClaimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim money from payment channel",
	Long:  "Increment payment channel nonce and send blockchain transaction to claim money from channel",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newClaimCommand)
	},
}

type claimCommand struct {
	channelService escrow.PaymentChannelService
	blockchain     *blockchain.Processor

	channelId *big.Int
	sendBack  bool
	timeout   time.Duration
}

func newClaimCommand(cmd *cobra.Command, args []string, components *Components) (command Command, err error) {
	channelId, err := getChannelId(cmd)
	if err != nil {
		return
	}
	timeout, err := time.ParseDuration(claimTimeout)
	if err != nil {
		return
	}

	command = &claimCommand{
		channelService: components.PaymentChannelService(),
		blockchain:     components.Blockchain(),

		channelId: channelId,
		sendBack:  claimSendBack,
		timeout:   timeout,
	}

	return
}

func getChannelId(cmd *cobra.Command) (id *big.Int, err error) {
	value := &big.Int{}
	err = value.UnmarshalText([]byte(claimChannelId))
	if err != nil {
		return nil, fmt.Errorf("Incorrect decimal number format: %v, error: %v", claimChannelId, err)
	}
	return value, nil
}

func (command *claimCommand) Run() (err error) {
	if !command.blockchain.Enabled() {
		return fmt.Errorf("blockchain should be enabled to claim money from channel")
	}

	var update escrow.ChannelUpdate
	if command.sendBack {
		update = escrow.CloseChannel
	} else {
		update = escrow.IncrementChannelNonce
	}

	claim, err := command.channelService.StartClaim(&escrow.PaymentChannelKey{ID: command.channelId}, update)
	if err != nil {
		return
	}

	err = command.claimPaymentFromChannel(claim)
	if err != nil {
		return
	}

	return
}

func (command *claimCommand) claimPaymentFromChannel(claim escrow.Claim) (err error) {
	payment := claim.Payment()

	err = command.blockchain.ClaimFundsFromChannel(
		command.timeout,
		payment.ChannelID,
		payment.Amount,
		payment.Signature,
		command.sendBack,
	)

	if err == nil {
		claim.Finish()
	}

	return
}
