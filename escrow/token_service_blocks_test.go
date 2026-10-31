package escrow

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/stretchr/testify/require"
)

func TestTokenServiceBlockWindow(t *testing.T) {
	for _, tt := range []struct {
		name  string
		block uint64
		valid bool
	}{
		{name: "current", block: 99, valid: true},
		{name: "past boundary", block: 94, valid: true},
		{name: "future boundary", block: 104, valid: true},
		{name: "stale", block: 93},
		{name: "future", block: 105},
		{name: "maximum uint64", block: ^uint64(0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newTokenServiceFixture(t)
			channel := f.channel()
			f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
				return channel, true, nil
			}
			transactions := 0
			f.channelMock.startPaymentTransactionFn = func(*Payment) (PaymentTransaction, error) {
				transactions++
				return &paymentTransactionMock{}, nil
			}
			f.usageMock.usage = map[string]*PrePaidData{
				USED_AMOUNT:    {Amount: big.NewInt(0)},
				PLANNED_AMOUNT: {Amount: big.NewInt(1)},
			}
			// Token freshness must use its own dependency, independently of payment validation.
			f.service.validator.blockValidator = blockchain.NewEthereumBlockValidator(func() (*big.Int, error) {
				return big.NewInt(500), nil
			})
			request := &TokenRequest{ChannelId: 1, SignedAmount: 11, CurrentBlock: tt.block}
			f.signRequest(request, f.senderKey)
			reply, err := f.service.GetToken(t.Context(), request)
			if tt.valid {
				require.NoError(t, err)
				require.NotNil(t, reply)
				require.Equal(t, "generated-token", reply.Token)
				require.Equal(t, 1, transactions)
				require.Equal(t, PLANNED_AMOUNT, f.usageMock.updatedType)
				require.True(t, f.tokenMock.created)
			} else {
				require.EqualError(t, err, "authentication failed as the signature passed has expired")
				require.Nil(t, reply)
				require.Zero(t, transactions)
				require.Empty(t, f.usageMock.updatedType)
				require.False(t, f.tokenMock.created)
			}
			require.Equal(t, big.NewInt(10), channel.AuthorizedAmount)
		})
	}
}
