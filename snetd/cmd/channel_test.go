package cmd

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/escrow"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChannelCommand(t *testing.T) {
	defer func() { paymentChannelId = "" }()

	t.Run("invalid channel id returns error", func(t *testing.T) {
		paymentChannelId = "not-a-number"
		command, err := newChannelCommand(nil, nil, &Components{})
		assert.Error(t, err)
		assert.Nil(t, command)
	})

	t.Run("valid channel id builds command", func(t *testing.T) {
		paymentChannelId = "42"

		command, err := newChannelCommand(nil, nil, &Components{
			etcdLockerStorage: storage.NewPrefixedAtomicStorage(storage.NewMemStorage(), "/payment-channel/lock"),
		})
		require.NoError(t, err)
		require.NotNil(t, command)

		channelCommand, ok := command.(*channelCommand)
		require.True(t, ok)
		assert.Equal(t, big.NewInt(42), channelCommand.paymentChannelId)
		assert.NotNil(t, channelCommand.storage)
	})
}

func TestChannelCommandRunNoChannelId(t *testing.T) {
	command := &channelCommand{}
	err := command.Run()
	assert.EqualError(t, err, "--unlock channel-id must be set")
}

func TestChannelCommandUnlock(t *testing.T) {
	t.Run("unlocks existing channel", func(t *testing.T) {
		paymentChannelId = "42"
		defer func() { paymentChannelId = "" }()

		command, err := newChannelCommand(nil, nil, &Components{
			etcdLockerStorage: storage.NewPrefixedAtomicStorage(storage.NewMemStorage(), "/payment-channel/lock"),
		})
		require.NoError(t, err)

		channelCommand, ok := command.(*channelCommand)
		require.True(t, ok)

		key := &escrow.PaymentChannelKey{ID: big.NewInt(42)}
		require.NoError(t, channelCommand.storage.Put(key.String(), "locked"))

		err = channelCommand.Run()
		assert.NoError(t, err)

		_, ok2, _ := channelCommand.storage.Get(key.String())
		assert.False(t, ok2, "channel lock should be removed")
	})

	t.Run("channel not found returns nil", func(t *testing.T) {
		paymentChannelId = "42"
		defer func() { paymentChannelId = "" }()

		command, err := newChannelCommand(nil, nil, &Components{
			etcdLockerStorage: storage.NewPrefixedAtomicStorage(storage.NewMemStorage(), "/payment-channel/lock"),
		})
		require.NoError(t, err)

		assert.NoError(t, command.Run(), "unlock of unknown channel should not return an error")
	})
}
