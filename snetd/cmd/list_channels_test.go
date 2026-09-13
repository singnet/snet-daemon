package cmd

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubPaymentChannelService is a mock of escrow.PaymentChannelService
type stubPaymentChannelService struct {
	channels    []*escrow.PaymentChannelData
	channelsErr error
	claims      []escrow.Claim
	claimsErr   error
}

func (s *stubPaymentChannelService) PaymentChannel(key *escrow.PaymentChannelKey) (*escrow.PaymentChannelData, bool, error) {
	return nil, false, nil
}
func (s *stubPaymentChannelService) ListChannels() ([]*escrow.PaymentChannelData, error) {
	return s.channels, s.channelsErr
}
func (s *stubPaymentChannelService) StartClaim(key *escrow.PaymentChannelKey, update escrow.ChannelUpdate) (escrow.Claim, error) {
	return nil, nil
}
func (s *stubPaymentChannelService) ListClaims() ([]escrow.Claim, error) {
	return s.claims, s.claimsErr
}
func (s *stubPaymentChannelService) StartPaymentTransaction(payment *escrow.Payment) (escrow.PaymentTransaction, error) {
	return nil, nil
}
func (s *stubPaymentChannelService) PaymentChannelFromBlockChain(key *escrow.PaymentChannelKey) (*escrow.PaymentChannelData, bool, error) {
	return nil, false, nil
}

func testPaymentChannel(id *big.Int) *escrow.PaymentChannelData {
	var sender common.Address
	_ = sender.UnmarshalText([]byte("0x671276c61943A35D5F230d076bDFd91B0c47bF09"))
	return &escrow.PaymentChannelData{
		ChannelID:        id,
		Nonce:            big.NewInt(1),
		State:            escrow.Open,
		Sender:           sender,
		Recipient:        sender,
		FullAmount:       big.NewInt(100),
		Expiration:       big.NewInt(1000),
		Signer:           sender,
		AuthorizedAmount: big.NewInt(10),
		Signature:        []byte("sig"),
	}
}

func TestNewListChannelsCommand(t *testing.T) {
	channelService := &stubPaymentChannelService{}
	components := &Components{paymentChannelService: channelService}

	command, err := newListChannelsCommand(nil, nil, components)
	require.NoError(t, err)
	require.NotNil(t, command)

	listCommand, ok := command.(*listChannelsCommand)
	require.True(t, ok)
	assert.Equal(t, channelService, listCommand.channelService)
}

func TestListChannelsCommandRun(t *testing.T) {
	t.Run("no channels prints empty message", func(t *testing.T) {
		command := &listChannelsCommand{channelService: &stubPaymentChannelService{}}
		assert.NoError(t, command.Run())
	})

	t.Run("lists all channels", func(t *testing.T) {
		channels := []*escrow.PaymentChannelData{
			testPaymentChannel(big.NewInt(1)),
			testPaymentChannel(big.NewInt(2)),
		}
		command := &listChannelsCommand{
			channelService: &stubPaymentChannelService{channels: channels},
		}
		assert.NoError(t, command.Run())
	})

	t.Run("service error is returned", func(t *testing.T) {
		expected := fmt.Errorf("list channels error")
		command := &listChannelsCommand{
			channelService: &stubPaymentChannelService{channelsErr: expected},
		}
		err := command.Run()
		assert.Equal(t, expected, err)
	})
}
