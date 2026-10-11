package escrow

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestAllowedUserHandlerRejections(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(metadata.MD)
		code   codes.Code
	}{
		{"channel ID", func(md metadata.MD) { md.Delete(handler.PaymentChannelIDHeader) }, codes.InvalidArgument},
		{"nonce", func(md metadata.MD) { md.Delete(handler.PaymentChannelNonceHeader) }, codes.InvalidArgument},
		{"amount", func(md metadata.MD) { md.Delete(handler.PaymentChannelAmountHeader) }, codes.InvalidArgument},
		{"invalid address", func(md metadata.MD) { md.Set(handler.PaymentMultiPartyEscrowAddressHeader, "invalid") }, codes.InvalidArgument},
		{"missing signature", func(md metadata.MD) { md.Delete(handler.PaymentChannelSignatureHeader) }, codes.InvalidArgument},
		{"invalid signature", func(md metadata.MD) { md.Set(handler.PaymentChannelSignatureHeader, "invalid") }, codes.Internal},
		{"curation", func(metadata.MD) { config.Vip().Set(config.AllowedUserFlag, true) }, codes.Internal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfig(t)
			key := GenerateTestPrivateKey()
			md := metadata.Pairs(
				handler.PaymentChannelIDHeader, "1", handler.PaymentChannelNonceHeader, "0",
				handler.PaymentChannelAmountHeader, "3",
				handler.PaymentMultiPartyEscrowAddressHeader, "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207",
				handler.PaymentChannelSignatureHeader, string(SignAllowedUserSignature(key)),
			)
			tt.change(md)
			h := AllowedUserPaymentHandler()
			require.Equal(t, EscrowPaymentType, h.Type())
			payment, err := h.Payment(&handler.GrpcStreamContext{MD: md})
			require.Nil(t, payment)
			require.NotNil(t, err)
			require.Equal(t, tt.code, err.Status.Code())
		})
	}
}

type channelProcessor struct {
	blockchain.Processor
	channel *blockchain.MultiPartyEscrowChannel
}

func (p channelProcessor) MultiPartyEscrowChannel(*big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
	return p.channel, true, nil
}

func TestBlockchainReaderAndValidatorConstructors(t *testing.T) {
	isolateConfig(t)
	f := newControlServiceFixture(t)
	org := f.service.organizationMetaData
	processor := channelProcessor{Processor: blockchain.NewMockProcessor(true), channel: &blockchain.MultiPartyEscrowChannel{
		Recipient: org.GetPaymentAddress(), Value: big.NewInt(30), Nonce: big.NewInt(1), Expiration: big.NewInt(50000),
	}}
	reader := NewBlockchainChannelReader(processor, config.Vip(), org)
	channel, ok, err := reader.GetChannelStateFromBlockchain(&PaymentChannelKey{ID: big.NewInt(7)})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, big.NewInt(7), channel.ChannelID)
	require.Equal(t, org.GetPaymentAddress(), channel.Recipient)
	require.Equal(t, big.NewInt(30), channel.FullAmount)
	validator := NewChannelPaymentValidator(processor, org)
	block, err := validator.currentBlock()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(blockchain.MockedCurrentBlock), block)
	require.Equal(t, org.GetPaymentExpirationThreshold(), validator.paymentExpirationThreshold())
}

func TestMergeChannelUsesNewerBlockchainNonce(t *testing.T) {
	local := &PaymentChannelData{Nonce: big.NewInt(1), AuthorizedAmount: big.NewInt(5)}
	chain := &PaymentChannelData{Nonce: big.NewInt(2), AuthorizedAmount: big.NewInt(0)}
	require.Same(t, chain, MergeStorageAndBlockchainChannelState(local, chain))
	require.Equal(t, big.NewInt(5), local.AuthorizedAmount)
}

func TestChannelStateRejectsStaleBlockAndStorageFailure(t *testing.T) {
	for _, failure := range []string{"stale block", "payment storage"} {
		t.Run(failure, func(t *testing.T) {
			isolateConfig(t)
			f := newControlServiceFixture(t)
			f.channelMock.paymentChannelFn = func(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
				return &PaymentChannelData{ChannelID: key.ID, Nonce: big.NewInt(1), Signer: f.signerAddr}, true, nil
			}
			f.channelMock.paymentChannelFromChainFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
				return &PaymentChannelData{Nonce: big.NewInt(0)}, true, nil
			}
			wantErr := errors.New("payment storage unavailable")
			payments := NewPaymentStorage(&failingStorage{MemoryStorage: storage.NewMemStorage(), getErr: wantErr})
			service := NewPaymentChannelStateService(f.channelMock, payments, blockchain.NewMockProcessor(true))
			request := &ChannelStateRequest{ChannelId: bigIntToBytes(big.NewInt(1)), CurrentBlock: 100}
			if failure == "stale block" {
				request.CurrentBlock = 94
			}
			request.Signature = getSignature(getChannelStateRequestMessage(common.Address{}, big.NewInt(1), new(big.Int).SetUint64(request.CurrentBlock)), f.signerKey)
			reply, err := service.GetChannelState(t.Context(), request)
			require.Nil(t, reply)
			if failure == "stale block" {
				require.ErrorContains(t, err, "signature passed has expired")
			} else {
				require.ErrorIs(t, err, wantErr)
			}
		})
	}
}

func TestPublishChannelStatsIgnoresOtherPaymentTypes(t *testing.T) {
	isolateConfig(t)
	config.Vip().Set(config.MeteringEnabled, true)
	err := PublishChannelStats(&Payment{}, func() (*big.Int, error) {
		t.Fatal("non-channel payment must not read blockchain or publish metrics")
		return nil, nil
	})
	require.Nil(t, err)
}
