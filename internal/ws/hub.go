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
	mu         sync.RWMutex
	rooms      map[int]map[*Client]bool
	register   chan *Client
	unregister chan *Client
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
		rooms:      make(map[int]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
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
			h.mu.Unlock()

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
			h.mu.Unlock()
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
