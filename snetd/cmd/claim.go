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
	Long: "Command gets latest payment from channel, moves payment to in-progress state" +
		" and writes payment transaction to the blockchain. Channel nonce is" +
		" updated and client should start using new nonce. User can specify --timeout" +
		" for blockchain writing. If payment was not written before timeout then writing" +
		" operation can be restarted using --payment-id option. See 'snetd list claims' to" +
		" list payments in progress.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAndCleanup(cmd, args, newClaimCommand)
	},
}

type claimCommand struct {
	channelService escrow.PaymentChannelService
	blockchain     *blockchain.Processor

	channelId *big.Int
	paymentId string
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
		paymentId: claimPaymentId,
		sendBack:  claimSendBack,
		timeout:   timeout,
	}

	return
}

func getChannelId(cmd *cobra.Command) (id *big.Int, err error) {
	if claimChannelId == "" {
		return nil, nil
	}
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
	if command.channelId == nil && command.paymentId == "" {
		return fmt.Errorf("either --channel-id or --payment-id flag should be set")
	}
	if command.channelId != nil && command.paymentId != "" {
		return fmt.Errorf("only one of --channel-id and --payment-id flags should be set")
	}

	if command.channelId != nil {
		return command.claimChannel()
	}
	if command.paymentId != "" {
		return command.claimPayment()
	}

	return
}

func (command *claimCommand) claimChannel() (err error) {
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

	return command.claimPaymentFromChannel(claim)
}

func (command *claimCommand) claimPayment() (err error) {
	claim, err := command.findClaim()
	if err != nil {
		return
	}

	return command.claimPaymentFromChannel(claim)
}

func (command *claimCommand) findClaim() (claim escrow.Claim, err error) {
	claims, err := command.channelService.ListClaims()
	if err != nil {
		return
	}

	for _, claim = range claims {
		if claim.Payment().ID() == command.paymentId {
			return
		}
	}

	return nil, fmt.Errorf("payment id is not found, id: %v", command.paymentId)
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
	if err != nil {
		return
	}

	return claim.Finish()
}
