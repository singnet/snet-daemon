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
}
