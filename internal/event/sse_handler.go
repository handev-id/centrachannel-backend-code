package event

import (
	"bufio"
	"encoding/json"
	"fmt"

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

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	client := h.broker.Subscribe(t.ID, uid)

	c.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer h.broker.Unsubscribe(client)

		initialData, _ := json.Marshal(map[string]int{"user_id": uid})
		if _, err := fmt.Fprintf(w, "event: connected\ndata: %s\n\n", initialData); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		for {
			select {
			case event, ok := <-client.ch:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, string(event.Data)); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-c.Context().Done():
				return
			}
		}
	})

	return nil
}
