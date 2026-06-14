package ws

import (
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"centrachannel/config"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type Handler struct {
	hub *Hub
	cfg *config.Config
}

func NewHandler(hub *Hub, cfg *config.Config) *Handler {
	return &Handler{hub: hub, cfg: cfg}
}

func (h *Handler) Handle(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	uid, err := middleware.GetUserID(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	upgrader := websocket.FastHTTPUpgrader{
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { return true },
	}

	_ = upgrader.Upgrade(c.RequestCtx(), func(conn *websocket.Conn) {
		client := &Client{
			hub:      h.hub,
			conn:     conn,
			tenantID: t.ID,
			userID:   uid,
			send:     make(chan []byte, 256),
		}
		h.hub.register <- client

		go client.writePump()
		client.readPump()
	})

	return nil
}
