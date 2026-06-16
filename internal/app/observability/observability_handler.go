package observability

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/event"
	"centrachannel/internal/utils/response"
)

type ObservabilityHandler struct {
	broker *event.SSEBroker
}

func NewObservabilityHandler(broker ...*event.SSEBroker) *ObservabilityHandler {
	h := &ObservabilityHandler{}
	if len(broker) > 0 {
		h.broker = broker[0]
	}
	return h
}

func (h *ObservabilityHandler) Health(c fiber.Ctx) error {
	return response.OK(c, "healthy", fiber.Map{"status": "ok"})
}

func (h *ObservabilityHandler) Ping(c fiber.Ctx) error {
	return c.SendString("pong")
}

func (h *ObservabilityHandler) Version(c fiber.Ctx) error {
	return response.OK(c, "ok", fiber.Map{"version": "1.0.0", "app": "CentraChannel API"})
}

func (h *ObservabilityHandler) SSETestPing(c fiber.Ctx) error {
	tenantIDStr := c.Query("tenant_id")
	if tenantIDStr == "" {
		return response.BadRequest(c, "tenant_id query param is required", nil)
	}
	tenantID, err := strconv.Atoi(tenantIDStr)
	if err != nil {
		return response.BadRequest(c, "tenant_id must be a number", nil)
	}
	h.broker.Notify(tenantID, "ping", map[string]string{"test": "true"})
	return response.OK(c, "ping sent", nil)
}
