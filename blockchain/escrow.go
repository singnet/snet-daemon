package blockchain

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"math/big"

	"go.uber.org/zap"
)

// MultiPartyEscrowChannel mirrors the MPE channels mapping. It contains on-chain
// state only, not the daemon's locally accepted cumulative payment authorization.
type MultiPartyEscrowChannel struct {
	Sender    common.Address
	Recipient common.Address
	// GroupId is the opaque replica-group identifier supplied when opening the
	// channel. The daemon compares it with organization metadata separately.
	GroupId [32]byte
	// Value is the current escrowed channel balance in token base units.
	Value *big.Int
	// Nonce identifies the current on-chain authorization state.
	Nonce *big.Int
	// Expiration is a block number, not a wall-clock timestamp.
	Expiration *big.Int
	Signer     common.Address
}

var zeroAddress = common.Address{}

// MultiPartyEscrowChannel reads the contract mapping by channel ID. A zero sender
// marks an absent entry; expiration, recipient/group ownership, and local payment
// eligibility are checked by higher-level escrow services.
func (processor *processor) MultiPartyEscrowChannel(channelID *big.Int) (channel *MultiPartyEscrowChannel, ok bool, err error) {
	channelIdField := zap.Any("channelID", channelID)

	ctx, cancel := processor.callContext()
	defer cancel()
	ch, err := processor.multiPartyEscrow.Channels(&bind.CallOpts{Context: ctx}, channelID)
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
