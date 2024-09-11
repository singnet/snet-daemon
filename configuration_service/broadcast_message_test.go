package configuration_service

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestNewChannelBroadcaster(t *testing.T) {

}

func TestChannelBroadcaster_NewSubscriber(t *testing.T) {
	broadcaster := NewChannelBroadcaster()
	assert.NotNil(t, broadcaster)
	channel1 := broadcaster.NewSubscriber()
	channel2 := broadcaster.NewSubscriber()
	assert.NotNil(t, channel1)
	assert.NotNil(t, channel2)
	//Add a message to trigger
	broadcaster.trigger <- 1
	msg1 := <-channel1
	msg2 := <-channel2
	assert.Equal(t, 1, msg1)
	//Check if all the subscribers received the same message
	assert.Equal(t, msg2, msg1)
	close(broadcaster.trigger)
}
