package cmd

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubClaim is a mock of escrow.Claim
type stubClaim struct {
	payment *escrow.Payment
}

func (c *stubClaim) Payment() *escrow.Payment { return c.payment }
func (c *stubClaim) Finish() error            { return nil }

func TestNewListClaimsCommand(t *testing.T) {
	channelService := &stubPaymentChannelService{}
	command, err := newListClaimsCommand(nil, nil, &Components{paymentChannelService: channelService})
	require.NoError(t, err)
	require.NotNil(t, command)

	listCommand, ok := command.(*listClaimsCommand)
	require.True(t, ok)
	assert.Equal(t, channelService, listCommand.channelService)
}

func TestListClaimsCommandRun(t *testing.T) {
	t.Run("no claims prints empty message", func(t *testing.T) {
		command := &listClaimsCommand{channelService: &stubPaymentChannelService{}}
		assert.NoError(t, command.Run())
	})

	t.Run("lists all claims", func(t *testing.T) {
		claims := []escrow.Claim{
			&stubClaim{payment: &escrow.Payment{ChannelID: big.NewInt(1), ChannelNonce: big.NewInt(2)}},
		}
		command := &listClaimsCommand{
			channelService: &stubPaymentChannelService{claims: claims},
		}
		assert.NoError(t, command.Run())
	})

	t.Run("service error is returned", func(t *testing.T) {
		expected := fmt.Errorf("list claims error")
		command := &listClaimsCommand{
			channelService: &stubPaymentChannelService{claimsErr: expected},
		}
		err := command.Run()
		assert.Equal(t, expected, err)
	})
}
