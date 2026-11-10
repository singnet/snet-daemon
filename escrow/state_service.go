//go:generate protoc -I . ./state_service.proto --go-grpc_out=paths=source_relative:. --go_out=paths=source_relative:.

package escrow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"go.uber.org/zap"
)

const (
	// AllowedBlockDifference is the inclusive freshness window for signed
	// state, provider-list, batch-claim, and prepaid-token requests.
	AllowedBlockDifference = 5
)

// PaymentChannelStateService exposes merged channel state to an authenticated
// channel signer, sender, or recipient, including a previous pending authorization.
type PaymentChannelStateService struct {
	channelService               PaymentChannelService
	paymentStorage               *PaymentStorage
	mpeAddress                   func() (address common.Address)
	compareWithLatestBlockNumber func(*big.Int) error
}

func (service *PaymentChannelStateService) mustEmbedUnimplementedPaymentChannelStateServiceServer() {
	// This marker satisfies the generated gRPC interface; RPC dispatch does not call it.
	panic("implement me")
}

// BlockChainDisabledStateService returns empty channel state without signature
// verification when blockchain integration is disabled.
type BlockChainDisabledStateService struct {
}

func (service *BlockChainDisabledStateService) mustEmbedUnimplementedPaymentChannelStateServiceServer() {
	// This marker satisfies the generated gRPC interface; RPC dispatch does not call it.
	panic("implement me")
}

func (service *BlockChainDisabledStateService) GetChannelState(context context.Context, request *ChannelStateRequest) (reply *ChannelStateReply, err error) {
	return &ChannelStateReply{}, nil
}

// StorageNonceMatchesWithBlockchainNonce compares the supplied channel snapshot's
// nonce with a direct blockchain read and reports an error if the read fails.
func (service *PaymentChannelStateService) StorageNonceMatchesWithBlockchainNonce(storageChannel *PaymentChannelData) (equal bool, err error) {
	h := service.channelService

	blockchainChannel, ok, err := h.PaymentChannelFromBlockChain(&PaymentChannelKey{ID: storageChannel.ChannelID})
	if err != nil {
		return false, errors.New("channel error:" + err.Error())
	}
	if !ok {
		return false, errors.New("unable to read channel details from blockchain.")
	}

	return storageChannel.Nonce.Cmp(blockchainChannel.Nonce) == 0, nil
}

// NewPaymentChannelStateService creates the authenticated channel-state service
// with the processor's MPE address and request-block freshness check.
func NewPaymentChannelStateService(channelService PaymentChannelService, paymentStorage *PaymentStorage, processor blockchain.Processor) *PaymentChannelStateService {
	return &PaymentChannelStateService{
		channelService: channelService,
		paymentStorage: paymentStorage,
		mpeAddress:     processor.EscrowContractAddress,
		compareWithLatestBlockNumber: func(blockNumberPassed *big.Int) error {
			return processor.CompareWithLatestBlockNumber(blockNumberPassed, AllowedBlockDifference)
		},
	}
}

// GetChannelState authenticates a signature over "__get_channel_state" || MPE
// address || channel ID || current block and returns the latest available state.
// The signer must be the channel signer, sender, or recipient, and the block must
// pass the inclusive AllowedBlockDifference freshness check.
//
// If the returned nonce differs from the blockchain nonce, the response includes
// the pending payment for current_nonce - 1. A missing payment is an error. This
// lookup assumes a previous pending claim; it does not validate that the nonce
// difference is exactly one. Prepaid usage fields are not populated here.
func (service *PaymentChannelStateService) GetChannelState(context context.Context, request *ChannelStateRequest) (reply *ChannelStateReply, err error) {
	zap.L().Debug("GetChannelState called",
		zap.Any("context", context),
		zap.Any("request", request))

	channelID := bytesToBigInt(request.GetChannelId())
	// Numeric signed fields use fixed-width, 32-byte big-endian encoding.
	message := bytes.Join([][]byte{
		[]byte("__get_channel_state"),
		service.mpeAddress().Bytes(),
		bigIntToBytes(channelID),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	signature := request.GetSignature()

	sender, err := utils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		return nil, errors.New("incorrect signature")
	}
	channel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelID})
	if err != nil {
		return nil, errors.New("channel error:" + err.Error())
	}
	if !ok {
		return nil, fmt.Errorf("channel is not found, channelId: %v", channelID)
	}

	if channel.Signer != *sender && *sender != channel.Sender && *sender != channel.Recipient {
		return nil, errors.New("only channel signer/sender/receiver can get latest channel state")
	}

	if err := service.compareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock))); err != nil {
		return nil, err
	}

	// A prepared claim can advance local state before its blockchain settlement.
	nonceEqual, err := service.StorageNonceMatchesWithBlockchainNonce(channel)
	if err != nil {
		zap.L().Info("payment data not available in payment storage.", zap.Error(err))
		return nil, err

	} else if !nonceEqual {
		// Return the previous nonce's pending authorization so the client can
		// account for it while the local and blockchain states differ.

		paymentID := PaymentID(channel.ChannelID, (&big.Int{}).Sub(channel.Nonce, big.NewInt(1)))
		payment, ok, err := service.paymentStorage.Get(paymentID)
		if err != nil {
			zap.L().Error("Error trying unable to extract old payment from storage", zap.Error(err))
			return nil, err
		}
		if !ok {
			zap.L().Error("old payment is not found in storage, nevertheless local channel nonce is not equal to the blockchain one", zap.Any("ChannelID", channelID))
			return nil, errors.New("channel has different nonce in local storage and blockchain and old payment is not found in storage")
		}
		return &ChannelStateReply{
			CurrentNonce:         bigIntToBytes(channel.Nonce),
			CurrentSignedAmount:  bigIntToBytes(channel.AuthorizedAmount),
			CurrentSignature:     channel.Signature,
			OldNonceSignedAmount: bigIntToBytes(payment.Amount),
			OldNonceSignature:    payment.Signature,
		}, nil
	}

	if channel.Signature == nil {
		return &ChannelStateReply{
			CurrentNonce: bigIntToBytes(channel.Nonce),
		}, nil
	}

	return &ChannelStateReply{
		CurrentNonce:        bigIntToBytes(channel.Nonce),
		CurrentSignedAmount: bigIntToBytes(channel.AuthorizedAmount),
		CurrentSignature:    channel.Signature,
	}, nil
}
