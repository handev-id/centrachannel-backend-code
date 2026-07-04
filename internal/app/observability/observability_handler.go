package observability

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/utils/response"
)

type ObservabilityHandler struct{}

func NewObservabilityHandler() *ObservabilityHandler {
	return &ObservabilityHandler{}
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
