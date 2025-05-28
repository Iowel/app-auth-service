package eventbus

import "sync"

const (
	EventType = "user.event"

	EventUserLogin    = "user.login"
	EventUserRegister = "user.register"
)

type Event struct {
	Type string
	Data any
}

type EventBus struct {
	subscribers []chan Event
	lock        sync.Mutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]chan Event, 0),
	}
}

func (e *EventBus) Publish(event Event) {
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, sub := range e.subscribers {
		select {
		case sub <- event:
		default:

		}
	}
}

func (e *EventBus) Subscribe() <-chan Event {
	ch := make(chan Event, 10)

	e.lock.Lock()
	defer e.lock.Unlock()

	e.subscribers = append(e.subscribers, ch)

	return ch 
}

func (e *EventBus) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()
	
	for _, sub := range e.subscribers {
		close(sub)
	}
}