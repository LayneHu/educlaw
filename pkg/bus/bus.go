package bus

import (
	"sync"
)

// MessageBus handles pub/sub messaging between components.
type MessageBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan OutboundMessage
}

// NewMessageBus creates a new MessageBus instance.
func NewMessageBus() *MessageBus {
	return &MessageBus{
		subscribers: make(map[string][]chan OutboundMessage),
	}
}

// Subscribe creates a new subscription channel for the given session ID.
func (mb *MessageBus) Subscribe(sessionID string) chan OutboundMessage {
	ch := make(chan OutboundMessage, 100)
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.subscribers[sessionID] = append(mb.subscribers[sessionID], ch)
	return ch
}

// Unsubscribe removes a subscription channel for the given session ID.
func (mb *MessageBus) Unsubscribe(sessionID string, ch chan OutboundMessage) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	subs := mb.subscribers[sessionID]
	for i, sub := range subs {
		if sub == ch {
			mb.subscribers[sessionID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
	if len(mb.subscribers[sessionID]) == 0 {
		delete(mb.subscribers, sessionID)
	}
}

// Publish sends a message to all subscribers of the given session.
func (mb *MessageBus) Publish(msg OutboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	subs := mb.subscribers[msg.SessionID]
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Drop message if channel is full
		}
	}
}

// HasSubscribers returns true if there are any subscribers for the session.
func (mb *MessageBus) HasSubscribers(sessionID string) bool {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return len(mb.subscribers[sessionID]) > 0
}
