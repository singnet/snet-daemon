//go:generate protoc -I . ./state_service.proto --go_out=plugins=grpc:.

package escrow

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/singnet/snet-daemon/authutils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"math/big"
)

// PaymentChannelStateService is an implementation of PaymentChannelStateServiceServer gRPC interface
type PaymentChannelStateService struct {
	channelService PaymentChannelService
	paymentStorage *PaymentStorage
}

// verifies whether storage channel nonce is equal to blockchain nonce or not
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

// NewPaymentChannelStateService returns new instance of PaymentChannelStateService
func NewPaymentChannelStateService(channelService PaymentChannelService, paymentStorage *PaymentStorage) *PaymentChannelStateService {
	return &PaymentChannelStateService{
		channelService: channelService,
		paymentStorage: paymentStorage,
	}
}

// GetChannelState returns the latest state of the channel which id is passed
// in request. To authenticate sender request should also contain correct
// signature of the channel id.
/*Simple case current_nonce == blockchain_nonce
unspent_amount = blockchain_value - current_signed_amount
Complex case current_nonce != blockchain_nonce
Taking into account our assumptions, we know that current_nonce = blockchain_nonce + 1.

unspent_amount = blockchain_value - oldnonce_signed_amount - current_signed_amount
It should be noted that in this case the server could send us smaller old nonce_signed_amount (not the actually last one which was used for channelClaim).
In this case, the server can only make us believe that we have more money in the channel then we actually have.
That means that one possible attack via unspent_amount is to make us believe that we have less tokens than we truly have,
and therefore reject future calls (or force us to call channelAddFunds).*/
func (service *PaymentChannelStateService) GetChannelState(context context.Context, request *ChannelStateRequest) (reply *ChannelStateReply, err error) {
	log.WithFields(log.Fields{
		"context": context,
		"request": request,
	}).Debug("GetChannelState called")

	channelID := bytesToBigInt(request.GetChannelId())
	channel, ok, err := service.channelService.PaymentChannel(&PaymentChannelKey{ID: channelID})
	if err != nil {
		return nil, errors.New("channel error:" + err.Error())
	}
	if !ok {
		return nil, fmt.Errorf("channel is not found, channelId: %v", channelID)
	}

	//For backward compatibility
	oldProto := false
	blockNumberPassed := int64(request.CurrentBlock)

	// signature verification
	message := bytes.Join([][]byte{
		[]byte("__get_channel_state"),
		channelID.Bytes(),
		abi.U256(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	signature := request.GetSignature()

	sender, err := authutils.GetSignerAddressFromMessage(message, signature)
	if err != nil {
		return nil, errors.New("incorrect signature")
	}

	//TODO remove this fall back to older signature versions. this is temporary, only to enable backward compatibility
	// with other components
	if channel.Signer != *sender {
		log.Infof("message does not follow the new signature standard. fall back to older signature standard")

		sender, err = authutils.GetSignerAddressFromMessage(bigIntToBytes(channelID), signature)
		if err != nil {
			return nil, errors.New("incorrect signature")
		}
		if channel.Signer != *sender {
			return nil, errors.New("only channel signer can get latest channel state")
		}
		if blockNumberPassed == 0 {
			oldProto = true
		}
	}

	if !oldProto {
		if err := authutils.CompareWithLatestBlockNumber(big.NewInt(int64(request.CurrentBlock))); err != nil {
			return nil, err
		}
	}

	// check if nonce matches with blockchain or not
	nonceEqual, err := service.StorageNonceMatchesWithBlockchainNonce(channel)
	if err != nil {
		log.WithError(err).Infof("payment data not available in payment storage.")
		return nil, err

	} else if !nonceEqual {
		// check for payments in the payment storage with current nonce - 1, this will happen  cli has issues in claiming process

		paymentID := PaymentID(channel.ChannelID, (&big.Int{}).Sub(channel.Nonce, big.NewInt(1)))
		payment, ok, err := service.paymentStorage.Get(paymentID)
		if err != nil {
			log.WithError(err).Errorf("Error trying unable to extract old payment from storage")
			return nil, err
		}
		if !ok {

			log.Errorf("old payment is not found in storage, nevertheless local channel nonce is not equal to the blockchain one, channel: %v", channelID)
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
