package license_server

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLicenseUsageTransaction(t *testing.T) {
	for _, usage := range []Usage{
		&UsageInAmount{UsageType: USED, Amount: big.NewInt(42)},
		&UsageInCalls{UsageType: USED, Calls: big.NewInt(3)},
	} {
		t.Run(usage.String(), func(t *testing.T) {
			channel := new(big.Int).Lsh(big.NewInt(1), 100)
			txn := licenseUsageTrackerTransactionImpl{channelId: channel, serviceId: "service", usage: usage}
			require.Equal(t, "service", txn.ServiceId())
			require.Equal(t, channel, txn.ChannelId())
			require.Same(t, usage, txn.Usage())
			before := new(big.Int).Set(usage.GetUsage())
			// Commit and Rollback currently have no persistence behavior.
			for range 2 {
				require.NoError(t, txn.Commit())
				require.NoError(t, txn.Rollback())
			}
			require.Equal(t, before, txn.Usage().GetUsage())
			require.Equal(t, "service", txn.ServiceId())
			require.Equal(t, channel, txn.ChannelId())
		})
	}
}

func TestLicenseUsageTransactionZeroValue(t *testing.T) {
	var txn licenseUsageTrackerTransactionImpl
	require.Empty(t, txn.ServiceId())
	require.Nil(t, txn.ChannelId())
	require.Nil(t, txn.Usage())
	require.NoError(t, txn.Commit())
	require.NoError(t, txn.Rollback())
}
