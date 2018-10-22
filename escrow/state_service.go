//go:generate protoc -I . ./state_service.proto --go_out=plugins=grpc:.
package escrow

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type PaymentChannelStateService struct {
	latest PaymentChannelStorage
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
		return nil, errors.New("incorrect signature")
	}

	channelId := bytesToBigInt(channelIdBytes)
	channel, ok, err := service.latest.Get(&PaymentChannelKey{ID: channelId})
	if err != nil {
		return nil, errors.New("channel storage error")
	}
	if !ok {
		return nil, fmt.Errorf("channel is not found, channelId: %v", channelId)
	}

	if channel.Sender != *sender {
		return nil, errors.New("only channel sender can get latest channel state")
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
