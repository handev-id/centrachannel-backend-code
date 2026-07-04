package ws

import (
	"encoding/json"
	"sync"
)

type Message struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type Client struct {
	TenantID int
	Send     chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[int]map[*Client]bool
	reg     chan *Client
	unreg   chan *Client
	broadcast chan broadcastReq
	done    chan struct{}
}

type broadcastReq struct {
	tenantID int
	data     []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:     make(map[int]map[*Client]bool),
		reg:       make(chan *Client, 256),
		unreg:     make(chan *Client, 256),
		broadcast: make(chan broadcastReq, 256),
		done:      make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.reg:
			h.mu.Lock()
			if h.rooms[client.TenantID] == nil {
				h.rooms[client.TenantID] = make(map[*Client]bool)
			}
			h.rooms[client.TenantID][client] = true
			h.mu.Unlock()

		case client := <-h.unreg:
			h.mu.Lock()
			if clients, ok := h.rooms[client.TenantID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.rooms, client.TenantID)
					}
				}
			}
			h.mu.Unlock()

		case req := <-h.broadcast:
			h.mu.RLock()
			clients := h.rooms[req.tenantID]
			h.mu.RUnlock()
			for client := range clients {
				select {
				case client.Send <- req.data:
				default:
				}
			}

		case <-h.done:
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) Register(client *Client) {
	h.reg <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unreg <- client
}

func (h *Hub) Broadcast(tenantID int, event string, data interface{}) {
	msg := Message{Event: event, Data: data}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.broadcast <- broadcastReq{tenantID: tenantID, data: raw}
}

type hubNotifier struct {
	hub *Hub
}

func NewHubNotifier(hub *Hub) Notifier {
	return &hubNotifier{hub: hub}
}

func (n *hubNotifier) Notify(tenantID int, event string, data interface{}) {
	n.hub.Broadcast(tenantID, event, data)
}
