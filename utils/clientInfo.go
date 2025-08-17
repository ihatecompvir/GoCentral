package utils

import (
	"errors"
	"sync"
)

type ClientInfo struct {
	IP       string
	PIDStack []uint32
	mu       sync.Mutex
}

type ClientStore struct {
	clients map[string]*ClientInfo
	mu      sync.RWMutex
}

var (
	clientStoreInstance *ClientStore
	once                sync.Once
)

func GetClientStoreSingleton() *ClientStore {
	once.Do(func() {
		clientStoreInstance = &ClientStore{
			clients: make(map[string]*ClientInfo),
		}
	})
	return clientStoreInstance
}

// how many PIDs can be stored in a client's stack
const maxPIDStack = 8

// adds a new client to the store
func (cs *ClientStore) AddClient(ip string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if _, exists := cs.clients[ip]; !exists {
		cs.clients[ip] = &ClientInfo{
			IP:       ip,
			PIDStack: make([]uint32, 0, maxPIDStack), // Limit stack size to maxPIDStack
		}
	}
}

// pushes a PID to the client's stack of PIDs (de-dupes and evicts oldest when full)
func (cs *ClientStore) PushPID(ip string, pid uint32) error {
	cs.mu.RLock()
	client, exists := cs.clients[ip]
	cs.mu.RUnlock()
	if !exists {
		return errors.New("client not found")
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	// If PID already exists, move it to the most-recent position (end) and return.
	for i, v := range client.PIDStack {
		if v == pid {
			// remove from current position
			copy(client.PIDStack[i:], client.PIDStack[i+1:])
			client.PIDStack = client.PIDStack[:len(client.PIDStack)-1]
			// append to end as most-recent
			client.PIDStack = append(client.PIDStack, pid)
			return nil
		}
	}

	// If full, evict the oldest (which will be idx 0 becausew this is a stack)
	if len(client.PIDStack) >= maxPIDStack {
		// drop oldest by shifting left one
		copy(client.PIDStack[0:], client.PIDStack[1:])
		client.PIDStack = client.PIDStack[:len(client.PIDStack)-1]
	}

	// Append new PID
	client.PIDStack = append(client.PIDStack, pid)
	return nil
}

// checks if a PID is valid for a client (i.e. have they logged in or called NintendoCreateAccount to switch to it, for multiple profile support)
func (cs *ClientStore) IsValidPID(ip string, pid uint32) (bool, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	client, exists := cs.clients[ip]
	if !exists {
		return false, errors.New("client not found")
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	for _, storedPID := range client.PIDStack {
		if storedPID == pid {
			return true, nil
		}
	}
	return false, nil
}

// removes a client from the store
func (cs *ClientStore) RemoveClient(ip string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.clients, ip)
}
