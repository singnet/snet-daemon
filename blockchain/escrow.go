package blockchain

import (
	"github.com/ethereum/go-ethereum/common"

	"math/big"

	"go.uber.org/zap"
)

type MultiPartyEscrowChannel struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    [32]byte
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
	Signer     common.Address
}

var zeroAddress = common.Address{}

func (processor *processor) MultiPartyEscrowChannel(channelID *big.Int) (channel *MultiPartyEscrowChannel, ok bool, err error) {
	channelIdField := zap.Any("channelID", channelID)

	ch, err := processor.multiPartyEscrow.Channels(nil, channelID)
	if err != nil {
		zap.L().Warn("Error while looking up for channel id in blockchain", zap.Error(err), channelIdField)
		return nil, false, err
	}
	if ch.Sender == zeroAddress {
		zap.L().Warn("Unable to find channel id in blockchain", channelIdField)
		return nil, false, nil
	}

	channel = &MultiPartyEscrowChannel{
		Sender:     ch.Sender,
		Recipient:  ch.Recipient,
		GroupId:    ch.GroupId,
		Value:      ch.Value,
		Nonce:      ch.Nonce,
		Expiration: ch.Expiration,
		Signer:     ch.Signer,
	}

	zap.L().Debug("Channel found in blockchain", zap.Any("channel", channel))

	return channel, true, nil
}
