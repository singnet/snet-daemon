package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"math/big"
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

func (processor *Processor) MultiPartyEscrowChannel(channelID *big.Int) (channel *MultiPartyEscrowChannel, ok bool, err error) {
	log := log.WithField("channelID", channelID)

	ch, err := processor.multiPartyEscrow.Channels(nil, channelID)
	if err != nil {
		log.WithError(err).Warn("Error while looking up for channel id in blockchain")
		return nil, false, err
	}
	if ch.Sender == zeroAddress {
		log.Warn("Unable to find channel id in blockchain")
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

	log = log.WithField("channel", channel)
	log.Debug("Channel found in blockchain")
	return channel, true, nil
}

func (processor *Processor) BlockChainChannelIds(sender *common.Address) (channelIds []*big.Int,
	ok bool, err error) {
	iterator,err  := processor.multiPartyEscrow.FilterChannelOpen(nil,[]common.Address{*sender},
	[]common.Address{processor.recipientAddress},[][32]byte{processor.groupId})

	if err != nil {
		log.WithError(err).Warn("Error while retrieving channels on blockchain")
		return nil, false, err
	}
	channelIds = make([]*big.Int,0)
	for iterator.Next() {
		channelIds = append(channelIds,iterator.Event.ChannelId)
	}
	return channelIds,true,nil
}