package configuration_service

import (
	"sync"
)

type MessageBroadcaster struct {
	//Operator UI can trigger changes to the Daemon configuration or  request the Daemon to stop/start  processing requests will ,
	// Hence we need a framework to receive this trigger and broadcast it to all the subscribers.
	trigger     chan int
	quit        chan int
	subscribers []chan int
	//This will be used to make sure we don't interfere with other threads
	mutex sync.Mutex
}

func NewChannelBroadcaster() *MessageBroadcaster {
	broadcaster := &MessageBroadcaster{}
	broadcaster.trigger = make(chan int, 1)
	go broadcaster.Publish()
	return broadcaster
}

// NewSubscriber Create a New Subscriber for this broadcaster message
// Interceptors or health checks  can subscribe to this and react accordingly
func (broadcast *MessageBroadcaster) NewSubscriber() chan int {
	ch := make(chan int, 1)
	broadcast.mutex.Lock()
	defer broadcast.mutex.Unlock()
	if broadcast.subscribers == nil {
		broadcast.subscribers = make([]chan int, 0)
	}
	broadcast.subscribers = append(broadcast.subscribers, ch)

	return ch
}

// Publish - Once a message is received, pass it down to all the subscribers
func (broadcast *MessageBroadcaster) Publish() {
	for {
		// Wait for the message to trigger the broadcast
		msg := <-broadcast.trigger
		broadcast.mutex.Lock()
		for _, subscriber := range broadcast.subscribers {
			// Now broad the message to all the subscribers.
			subscriber <- msg
		}
		broadcast.mutex.Unlock()
	}
}
