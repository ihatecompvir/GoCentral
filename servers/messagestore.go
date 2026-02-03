package servers

import (
	"log"
	"rb3server/serialization/message"
	"sync"
	"sync/atomic"
	"time"
)

// StoredMessage wraps a TextMessage with its expiration time
type StoredMessage struct {
	Message    message.TextMessage
	ExpiresAt  time.Time
	ReceivedAt time.Time
}

// MessageStore holds messages in memory with automatic expiration
type MessageStore struct {
	mu            sync.RWMutex
	messages      map[uint32][]StoredMessage // keyed by recipient PID
	quit          chan struct{}
	nextMessageID uint32 // atomic counter for unique message IDs
}

var GlobalMessageStore *MessageStore

// InitMessageStore creates and starts the global message store
func InitMessageStore() {
	GlobalMessageStore = &MessageStore{
		messages: make(map[uint32][]StoredMessage),
		quit:     make(chan struct{}),
	}
	go GlobalMessageStore.purgeLoop()
	log.Println("In-memory message store initialized")
}

// StopMessageStore stops the message store's purge loop
func StopMessageStore() {
	if GlobalMessageStore != nil {
		close(GlobalMessageStore.quit)
	}
}

// NextMessageID returns the next unique message ID (thread-safe)
func (ms *MessageStore) NextMessageID() uint32 {
	return atomic.AddUint32(&ms.nextMessageID, 1)
}

// AddMessage stores a message for the specified recipient
func (ms *MessageStore) AddMessage(recipientPID uint32, msg message.TextMessage) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	// LifeTime is in seconds
	expiresAt := now.Add(time.Duration(msg.LifeTime) * time.Second)

	stored := StoredMessage{
		Message:    msg,
		ExpiresAt:  expiresAt,
		ReceivedAt: now,
	}

	ms.messages[recipientPID] = append(ms.messages[recipientPID], stored)
	log.Printf("Stored message for PID %d, expires at %v (lifetime: %d seconds)\n",
		recipientPID, expiresAt, msg.LifeTime)
}

// GetMessages retrieves all non-expired messages for a recipient
func (ms *MessageStore) GetMessages(recipientPID uint32) []message.TextMessage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	stored, exists := ms.messages[recipientPID]
	if !exists {
		return nil
	}

	now := time.Now()
	var result []message.TextMessage
	for _, sm := range stored {
		if now.Before(sm.ExpiresAt) {
			result = append(result, sm.Message)
		}
	}

	return result
}

// GetAndClearMessages retrieves all non-expired messages for a recipient and removes them
func (ms *MessageStore) GetAndClearMessages(recipientPID uint32) []message.TextMessage {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	stored, exists := ms.messages[recipientPID]
	if !exists {
		return nil
	}

	now := time.Now()
	var result []message.TextMessage
	for _, sm := range stored {
		if now.Before(sm.ExpiresAt) {
			result = append(result, sm.Message)
		}
	}

	// Clear messages for this recipient after retrieval
	delete(ms.messages, recipientPID)

	return result
}

// purgeLoop periodically removes expired messages
func (ms *MessageStore) purgeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.purgeExpired()
		case <-ms.quit:
			return
		}
	}
}

// GetMessagesByIDs retrieves specific messages by their IDs for a recipient
func (ms *MessageStore) GetMessagesByIDs(recipientPID uint32, messageIDs []uint32, deleteAfter bool) []message.TextMessage {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	stored, exists := ms.messages[recipientPID]
	if !exists {
		return nil
	}

	// Build a set of requested IDs for fast lookup
	idSet := make(map[uint32]bool)
	for _, id := range messageIDs {
		idSet[id] = true
	}

	now := time.Now()
	var result []message.TextMessage
	var remaining []StoredMessage

	for _, sm := range stored {
		if now.Before(sm.ExpiresAt) {
			if idSet[sm.Message.ID] {
				result = append(result, sm.Message)
				if !deleteAfter {
					remaining = append(remaining, sm)
				}
			} else {
				remaining = append(remaining, sm)
			}
		}
	}

	if deleteAfter {
		if len(remaining) == 0 {
			delete(ms.messages, recipientPID)
		} else {
			ms.messages[recipientPID] = remaining
		}
	}

	return result
}

// DeleteMessages removes specific messages by their IDs for a recipient
func (ms *MessageStore) DeleteMessages(recipientPID uint32, messageIDs []uint32) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	stored, exists := ms.messages[recipientPID]
	if !exists {
		return
	}

	// Build a set of IDs to delete for fast lookup
	idSet := make(map[uint32]bool)
	for _, id := range messageIDs {
		idSet[id] = true
	}

	var remaining []StoredMessage
	deletedCount := 0

	for _, sm := range stored {
		if !idSet[sm.Message.ID] {
			remaining = append(remaining, sm)
		} else {
			deletedCount++
		}
	}

	if len(remaining) == 0 {
		delete(ms.messages, recipientPID)
	} else {
		ms.messages[recipientPID] = remaining
	}

	if deletedCount > 0 {
		log.Printf("Deleted %d messages for PID %d\n", deletedCount, recipientPID)
	}
}

// purgeExpired removes all expired messages from the store
func (ms *MessageStore) purgeExpired() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	purgedCount := 0

	for pid, stored := range ms.messages {
		var valid []StoredMessage
		for _, sm := range stored {
			if now.Before(sm.ExpiresAt) {
				valid = append(valid, sm)
			} else {
				purgedCount++
			}
		}

		if len(valid) == 0 {
			delete(ms.messages, pid)
		} else {
			ms.messages[pid] = valid
		}
	}

	if purgedCount > 0 {
		log.Printf("Purged %d expired messages from message store\n", purgedCount)
	}
}
