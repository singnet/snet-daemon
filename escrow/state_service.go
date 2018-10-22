//go:generate protoc -I . ./state_service.proto --go_out=plugins=grpc:.
package escrow

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

type PaymentChannelStateService struct {
	latest LatestPaymentStorage
	closed ClosedPaymentStorage
}

func (service *PaymentChannelStateService) GetChannelState(context context.Context, request *ChannelStateRequest) (reply *ChannelStateReply, err error) {
	log.WithFields(log.Fields{
		"context": context,
		"request": request,
	}).Debug("GetChannelState called")

	channelIdBytes := request.GetChannelId()
	signature := request.GetSignature()
	sender, err := getSignerAddressFromMessage(channelIdBytes, signature)
	if err != nil {
		return nil, status.New(codes.Unauthenticated, "Incorrect signature")
	}

	channelId := bytesToBigInt(channelIdBytes)
	channel, ok, err := service.latest.GetByID(channelId)
	if err != nil {
		return nil, status.New(codes.Internal, "Channel storage error")
	}
	if !ok {
		return nil, status.Newf(codes.NotFound, "Channel is not found, channelId: %v", channelId)
	}

	if channel.Sender != sender {
		return nil, status.New(codes.Unauthenticated, "Only channel sender can get latest channel state")
	}

	if channel.Signature != nil {
		return &ChannelStateReply{
			CurrentNonce: channel.Nonce,
			CurrentValue: channel.FullAmount,
			SignedNonce:  signed.Nonce,
			SignedAmount: signed.AuthorizedAmount,
			Signature:    signed.Signature,
		}, nil
	}

	if channel.Nonce.Cmp(big.NewInt(0)) == 0 {
		return &ChannelStateReply{
			CurrentNonce: channel.Nonce,
			CurrentValue: channel.FullAmount,
		}, nil
	}

	prevNonce := (&big.Int{}).Sub(channel.Nonce, big.NewInt(1))
	prevChannel, ok, err := service.closed.GetByIDAndNonce(channelId, prevNonce)
	if err != nil {
		return nil, status.New(codes.Internal, "Channel storage error")
	}
	if !ok {
		return &ChannelStateReply{
			CurrentNonce: channel.Nonce,
			CurrentValue: channel.FullAmount,
		}, nil
	}

	return &ChannelStateReply{
		CurrentNonce: channel.Nonce,
		CurrentValue: channel.FullAmount,
		SignedNonce:  prevChannel.Nonce,
		SignedAmount: prevChannel.AuthorizedAmount,
		Signature:    prevChannel.Signature,
	}, nil
}
