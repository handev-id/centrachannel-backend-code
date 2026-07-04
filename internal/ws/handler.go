package ws

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	fasthttpws "github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"

	"centrachannel/internal/utils/logger"
)

const (
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxMessageSize    = 512
	TokenCookieName   = "CrmToken"
)

type WSHandler struct {
	hub       *Hub
	jwtSecret string
	logger    *logger.Logger
}

func NewWSHandler(hub *Hub, jwtSecret string, log *logger.Logger) *WSHandler {
	return &WSHandler{hub: hub, jwtSecret: jwtSecret, logger: log}
}

var upgrader = fasthttpws.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func (h *WSHandler) Handle(c fiber.Ctx) error {
	tokenStr := h.extractToken(c)
	if tokenStr == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	tenantID, err := h.validateToken(tokenStr)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	err = upgrader.Upgrade(c.RequestCtx(), func(wsConn *fasthttpws.Conn) {
		client := &Client{
			TenantID: tenantID,
			Send:     make(chan []byte, 256),
		}

		h.hub.Register(client)
		defer h.hub.Unregister(client)

		go h.writePump(wsConn, client)
		h.readPump(wsConn, client)
	})

	if err != nil {
		h.logger.Warn("ws: upgrade failed: %v", err)
		return nil
	}

	return nil
}

func (h *WSHandler) extractToken(c fiber.Ctx) string {
	req := c.RequestCtx()
	tokenStr := string(req.Request.Header.Cookie(TokenCookieName))
	if tokenStr == "" {
		tokenStr = string(req.Request.Header.Peek("Authorization"))
	}
	if tokenStr == "" {
		tokenStr = c.Query("token")
	}
	if len(tokenStr) > 7 && strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = tokenStr[7:]
	}
	return tokenStr
}

func (h *WSHandler) validateToken(tokenStr string) (int, error) {
	token, err := jwt.Parse(tokenStr, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, nil
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, fiber.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fiber.ErrUnauthorized
	}

	tenantFloat, ok := claims["tenant"].(float64)
	if !ok {
		return 0, fiber.ErrUnauthorized
	}

	return int(tenantFloat), nil
}

func (h *WSHandler) readPump(c *fasthttpws.Conn, client *Client) {
	defer c.Close()

	c.SetReadLimit(maxMessageSize)
	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *WSHandler) writePump(c *fasthttpws.Conn, client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			c.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.WriteMessage(fasthttpws.CloseMessage, []byte{})
				return
			}
			if err := c.WriteMessage(fasthttpws.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.WriteMessage(fasthttpws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
