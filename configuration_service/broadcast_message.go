package configuration_service

import (
	"context"
	"sync"
)

type MessageBroadcaster struct {
	//Operator UI can trigger changes to the Daemon configuration or  request the Daemon to stop/start  processing requests will ,
	// Hence we need a framework to receive this trigger and broadcast it to all the subscribers.
	trigger     chan int
	quit        chan struct{}
	done        chan struct{}
	closeOnce   sync.Once
	closed      bool
	subscribers []chan int
	//This will be used to make sure we don't interfere with other threads
	mutex sync.Mutex
}

func NewChannelBroadcaster() *MessageBroadcaster {
	broadcaster := &MessageBroadcaster{}
	broadcaster.trigger = make(chan int, 1)
	broadcaster.quit = make(chan struct{})
	broadcaster.done = make(chan struct{})
	go broadcaster.Publish()
	return broadcaster
}

// NewSubscriber Create a New Subscriber for this broadcaster message
// Interceptors or health checks  can subscribe to this and react accordingly
func (broadcast *MessageBroadcaster) NewSubscriber() chan int {
	ch := make(chan int, 1)
	broadcast.mutex.Lock()
	defer broadcast.mutex.Unlock()
	if broadcast.closed {
		close(ch)
		return ch
	}
	if broadcast.subscribers == nil {
		broadcast.subscribers = make([]chan int, 0)
	}
	broadcast.subscribers = append(broadcast.subscribers, ch)

	return ch
}

// Publish - Once a message is received, pass it down to all the subscribers
func (broadcast *MessageBroadcaster) Publish() {
	defer close(broadcast.done)
	defer broadcast.closeSubscribers()
	for {
		select {
		case <-broadcast.quit:
			return
		case msg, ok := <-broadcast.trigger:
			if !ok {
				return
			}
			broadcast.mutex.Lock()
			for _, subscriber := range broadcast.subscribers {
				// Only the latest state matters; a slow subscriber must not block shutdown.
				select {
				case <-subscriber:
				default:
				}
				subscriber <- msg
			}
			broadcast.mutex.Unlock()
		}
	}
}

func (broadcast *MessageBroadcaster) Send(ctx context.Context, value int) bool {
	broadcast.mutex.Lock()
	closed := broadcast.closed
	broadcast.mutex.Unlock()
	if closed {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case <-broadcast.quit:
		return false
	case broadcast.trigger <- value:
		return true
	}
}

func (broadcast *MessageBroadcaster) closeSubscribers() {
	broadcast.mutex.Lock()
	defer broadcast.mutex.Unlock()
	broadcast.closed = true
	for _, subscriber := range broadcast.subscribers {
		close(subscriber)
	}
	broadcast.subscribers = nil
}

func (broadcast *MessageBroadcaster) Close() {
	broadcast.closeOnce.Do(func() {
		if broadcast.quit != nil {
			close(broadcast.quit)
		} else {
			broadcast.closeSubscribers()
		}
	})
	if broadcast.done != nil {
		<-broadcast.done
	}
}
