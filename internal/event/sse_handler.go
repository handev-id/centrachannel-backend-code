package event

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type Handler struct {
	broker *SSEBroker
}

func NewHandler(broker *SSEBroker) *Handler {
	return &Handler{broker: broker}
}

func (h *Handler) Handle(c fiber.Ctx, t *tenant.Tenant) error {
	uid, err := middleware.GetUserID(c)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}

	origin := c.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	fctx := c.RequestCtx()
	fctx.HijackSetNoResponse(true)
	fctx.Hijack(func(cw net.Conn) {
		defer cw.Close()

		cw.Write([]byte("HTTP/1.1 200 OK\r\n"))
		cw.Write([]byte("Content-Type: text/event-stream\r\n"))
		cw.Write([]byte("Cache-Control: no-cache\r\n"))
		cw.Write([]byte("Connection: keep-alive\r\n"))
		cw.Write([]byte("X-Accel-Buffering: no\r\n"))
		cw.Write([]byte("Access-Control-Allow-Origin: " + origin + "\r\n"))
		cw.Write([]byte("\r\n"))

		client := h.broker.Subscribe(t.ID, uid)
		defer h.broker.Unsubscribe(client)

		initialData, _ := json.Marshal(map[string]int{"user_id": uid})
		fmt.Fprintf(cw, "event: connected\ndata: %s\n\n", string(initialData))

		for {
			select {
			case event, ok := <-client.ch:
				if !ok {
					return
				}
				fmt.Fprintf(cw, "event: %s\ndata: %s\n\n", event.Event, string(event.Data))
			case <-c.Context().Done():
				return
			}
		}
	})

	return nil
}
