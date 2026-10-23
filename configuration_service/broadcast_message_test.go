package configuration_service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"time"

	"testing"
)

func TestBroadcasterCloseDoesNotWaitForSlowSubscribers(t *testing.T) {
	b := NewChannelBroadcaster()
	b.NewSubscriber()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	for i := 0; i < 10; i++ {
		if !b.Send(ctx, i) {
			t.Fatal("publisher blocked behind a slow subscriber")
		}
	}
	done := make(chan struct{})
	go func() { b.Close(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("broadcaster did not stop")
	}
	if _, ok := <-b.NewSubscriber(); ok {
		t.Fatal("subscriber was admitted after close")
	}
	if b.Send(ctx, 1) {
		t.Fatal("message was admitted after close")
	}
	b.Close()
}

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
