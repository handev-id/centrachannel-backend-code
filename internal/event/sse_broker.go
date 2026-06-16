package event

import (
	"encoding/json"
	"sync"
	"time"
)

type SSEBroker struct {
	mu          sync.RWMutex
	rooms       map[int]map[*SSEClient]bool
	onlineUsers map[int]map[int]int
}

type SSEClient struct {
	ch       chan SSEEvent
	tenantID int
	userID   int
}

type SSEEvent struct {
	Event string
	Data  []byte
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		rooms:       make(map[int]map[*SSEClient]bool),
		onlineUsers: make(map[int]map[int]int),
	}
}

func (b *SSEBroker) Subscribe(tenantID, userID int) *SSEClient {
	client := &SSEClient{
		ch:       make(chan SSEEvent, 256),
		tenantID: tenantID,
		userID:   userID,
	}

	b.mu.Lock()

	if b.rooms[tenantID] == nil {
		b.rooms[tenantID] = make(map[*SSEClient]bool)
	}
	b.rooms[tenantID][client] = true

	if b.onlineUsers[tenantID] == nil {
		b.onlineUsers[tenantID] = make(map[int]int)
	}

	b.onlineUsers[tenantID][userID]++
	firstConnection := b.onlineUsers[tenantID][userID] == 1
	b.mu.Unlock()

	if firstConnection {
		b.Notify(tenantID, "user:online", map[string]int{"id": userID})
	}

	return client
}

func (b *SSEBroker) Unsubscribe(client *SSEClient) {
	b.mu.Lock()
	if clients, ok := b.rooms[client.tenantID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.ch)
			if len(clients) == 0 {
				delete(b.rooms, client.tenantID)
			}
		}
	}

	lastDisconnect := false
	if counts, ok := b.onlineUsers[client.tenantID]; ok {
		counts[client.userID]--
		if counts[client.userID] <= 0 {
			delete(counts, client.userID)
			if len(counts) == 0 {
				delete(b.onlineUsers, client.tenantID)
			}
			lastDisconnect = true
		}
	}
	b.mu.Unlock()

	if lastDisconnect {
		b.Notify(client.tenantID, "user:offline", map[string]int{"id": client.userID})
	}
}

func (b *SSEBroker) Notify(tenantID int, event string, data interface{}) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for client := range b.rooms[tenantID] {
		select {
		case client.ch <- SSEEvent{Event: event, Data: msg}:
		default:
		}
	}
}

func (b *SSEBroker) StartHeartbeat() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			b.mu.RLock()
			tenants := make([]int, 0, len(b.rooms))
			for tenantID := range b.rooms {
				tenants = append(tenants, tenantID)
			}
			b.mu.RUnlock()

			for _, tenantID := range tenants {
				b.Notify(tenantID, "ping", map[string]string{})
			}
		}
	}()
}

func (b *SSEBroker) GetOnlineUsers(tenantID int) []int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var users []int
	if counts, ok := b.onlineUsers[tenantID]; ok {
		for userID := range counts {
			users = append(users, userID)
		}
	}
	return users
}

func (b *SSEBroker) IsUserOnline(tenantID int, userID int) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if counts, ok := b.onlineUsers[tenantID]; ok {
		_, ok := counts[userID]
		return ok
	}
	return false
}
