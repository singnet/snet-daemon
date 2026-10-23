package freecall

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisabledStateService(t *testing.T) {
	ctx := context.Background()

	freeCall := &DisabledStateService{}
	freeCallToken, err := freeCall.GetFreeCallToken(ctx, &GetFreeCallTokenRequest{})
	require.NotNil(t, freeCallToken)
	require.Empty(t, freeCallToken.GetToken())
	require.EqualError(t, err, "error in generating token because blockchain is disabled, contact service provider")

	freeCallReply, err := freeCall.GetFreeCallsAvailable(ctx, &FreeCallStateRequest{})
	require.Equal(t, uint64(0), freeCallReply.GetFreeCallsAvailable())
	require.EqualError(t, err, "error in determining free calls because blockchain is disabled, contact service provider")
}
