package escrow

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBlockchainDisabledServicesReturnPredictableReplies(t *testing.T) {
	ctx := context.Background()

	control := &BlockChainDisabledProviderControlService{}
	unclaimed, err := control.GetListUnclaimed(ctx, &GetPaymentsListRequest{})
	require.NoError(t, err)
	require.Empty(t, unclaimed.GetPayments())

	inProgress, err := control.GetListInProgress(ctx, &GetPaymentsListRequest{})
	require.NoError(t, err)
	require.Empty(t, inProgress.GetPayments())

	claim, err := control.StartClaim(ctx, &StartClaimRequest{})
	require.NoError(t, err)
	require.NotNil(t, claim)

	multipleClaims, err := control.StartClaimForMultipleChannels(ctx, &StartMultipleClaimRequest{})
	require.NoError(t, err)
	require.Empty(t, multipleClaims.GetPayments())

	state := &BlockChainDisabledStateService{}
	channel, err := state.GetChannelState(ctx, &ChannelStateRequest{})
	require.NoError(t, err)
	require.Zero(t, channel.GetPlannedAmount())

	token := BlockChainDisabledTokenService{}
	tokenReply, err := token.GetToken(ctx, &TokenRequest{})
	require.NoError(t, err)
	require.Empty(t, tokenReply.GetToken())

	freeCall := &BlockChainDisabledFreeCallStateService{}
	freeCallToken, err := freeCall.GetFreeCallToken(ctx, &GetFreeCallTokenRequest{})
	require.NotNil(t, freeCallToken)
	require.Empty(t, freeCallToken.GetToken())
	require.EqualError(t, err, "error in generating token because blockchain is disabled, contact service provider")

	freeCallReply, err := freeCall.GetFreeCallsAvailable(ctx, &FreeCallStateRequest{})
	require.Equal(t, uint64(0), freeCallReply.GetFreeCallsAvailable())
	require.EqualError(t, err, "error in determining free calls because blockchain is disabled, contact service provider")
}

func TestPaymentChannelValueObjectsAndUpdates(t *testing.T) {
	payment := &Payment{
		MpeContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000ab"),
		ChannelID:          big.NewInt(7),
		ChannelNonce:       big.NewInt(2),
		Amount:             big.NewInt(11),
		Signature:          []byte{1, 2, 3},
	}
	require.Equal(t, "7/2", payment.ID())
	require.Equal(t, "7/2", PaymentID(big.NewInt(7), big.NewInt(2)))
	require.Contains(t, payment.String(), "ChannelID: 7")
	require.Equal(t, "{ID: 7}", (&PaymentChannelKey{ID: big.NewInt(7)}).String())
	require.Equal(t, "Open", Open.String())
	require.Equal(t, "Closed", Closed.String())

	channel := &PaymentChannelData{
		ChannelID:        big.NewInt(7),
		Nonce:            big.NewInt(2),
		FullAmount:       big.NewInt(20),
		AuthorizedAmount: big.NewInt(6),
		Signature:        []byte{1},
	}
	CloseChannel(channel)
	require.Zero(t, channel.FullAmount.Sign())

	channel.FullAmount.SetInt64(20)
	IncrementChannelNonce(channel)
	require.Equal(t, int64(3), channel.Nonce.Int64())
	require.Equal(t, int64(14), channel.FullAmount.Int64())
	require.Zero(t, channel.AuthorizedAmount.Sign())
	require.Nil(t, channel.Signature)
	require.True(t, strings.Contains(channel.String(), "ChannelID: 7"))
}

func TestPrepaidValueObjectsAndTransaction(t *testing.T) {
	transaction := prePaidTransactionImpl{
		channelId: big.NewInt(9),
		price:     big.NewInt(17),
		signer:    common.HexToAddress("0x00000000000000000000000000000000000000cd"),
	}
	require.Equal(t, int64(9), transaction.ChannelId().Int64())
	require.Equal(t, int64(17), transaction.Price().Int64())
	require.Equal(t, transaction.signer, transaction.GetSender())
	require.NoError(t, transaction.Commit())
	require.NoError(t, transaction.Rollback())

	payment := &PrePaidPayment{ChannelID: big.NewInt(9), OrganizationId: "org", GroupId: "group"}
	require.Equal(t, "{ID:9/org/group}", payment.String())

	key := &PrePaidDataKey{ChannelID: big.NewInt(9), UsageType: USED_AMOUNT}
	require.Equal(t, "{ID:9/U}", key.String())
	require.Equal(t, "{Amount:17}", (&PrePaidData{Amount: big.NewInt(17)}).String())

	usage := &PrePaidUsageData{
		ChannelID:       big.NewInt(9),
		PlannedAmount:   big.NewInt(1),
		UsedAmount:      big.NewInt(2),
		RefundAmount:    big.NewInt(3),
		UpdateUsageType: USED_AMOUNT,
	}
	require.Contains(t, usage.String(), "ChannelID:9")
}
