package escrow

import (
	"context"
	"testing"

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
