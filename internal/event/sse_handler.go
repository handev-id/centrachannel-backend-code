package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"

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
	log.Printf("[sse] Handle called: uid=%d tid=%d", uid, t.ID)

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	streamCtx := c.Context()
	c.Abandon()

	return c.SendStreamWriter(func(w *bufio.Writer) {
		log.Printf("[sse] StreamWriter invoked: uid=%d tid=%d", uid, t.ID)
		client := h.broker.Subscribe(t.ID, uid)
		defer h.broker.Unsubscribe(client)

		initialData, _ := json.Marshal(map[string]int{"user_id": uid})
		fmt.Fprintf(w, "event: connected\ndata: %s\n\n", string(initialData))
		w.Flush()

		for {
			select {
			case <-streamCtx.Done():
				log.Printf("[sse] Client disconnected: uid=%d", uid)
				return
			case event, ok := <-client.ch:
				if !ok {
					log.Printf("[sse] Channel closed: uid=%d", uid)
					return
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, string(event.Data))
				w.Flush()
			}
		}
	})
}
