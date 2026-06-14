package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Hub struct {
	mu          sync.RWMutex
	rooms       map[int]map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	onlineUsers map[int]map[int]int
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	tenantID int
	userID   int
	send     chan []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:       make(map[int]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		onlineUsers: make(map[int]map[int]int),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.tenantID] == nil {
				h.rooms[client.tenantID] = make(map[*Client]bool)
			}
			h.rooms[client.tenantID][client] = true

			if h.onlineUsers[client.tenantID] == nil {
				h.onlineUsers[client.tenantID] = make(map[int]int)
			}
			h.onlineUsers[client.tenantID][client.userID]++
			firstConnection := h.onlineUsers[client.tenantID][client.userID] == 1
			tenantID := client.tenantID
			userID := client.userID
			h.mu.Unlock()

			if firstConnection {
				h.Notify(tenantID, "user:online", map[string]int{"id": userID})
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.tenantID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.tenantID)
					}
				}
			}

			lastDisconnect := false
			tenantID := client.tenantID
			userID := client.userID
			if counts, ok := h.onlineUsers[tenantID]; ok {
				counts[userID]--
				if counts[userID] <= 0 {
					delete(counts, userID)
					if len(counts) == 0 {
						delete(h.onlineUsers, tenantID)
					}
					lastDisconnect = true
				}
			}
			h.mu.Unlock()

			if lastDisconnect {
				h.Notify(tenantID, "user:offline", map[string]int{"id": userID})
			}
		}
	}
}

func (h *Hub) Notify(tenantID int, event string, data interface{}) {
	msg, err := json.Marshal(map[string]interface{}{
		"event": event,
		"data":  data,
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := h.rooms[tenantID]
	h.mu.RUnlock()

	if clients == nil {
		return
	}

	for client := range clients {
		select {
		case client.send <- msg:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) GetOnlineUsers(tenantID int) []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var users []int
	if counts, ok := h.onlineUsers[tenantID]; ok {
		for userID := range counts {
			users = append(users, userID)
		}
	}
	return users
}

func (h *Hub) IsUserOnline(tenantID int, userID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if counts, ok := h.onlineUsers[tenantID]; ok {
		_, ok := counts[userID]
		return ok
	}
	return false
}

func (c *Client) readPump() {
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
