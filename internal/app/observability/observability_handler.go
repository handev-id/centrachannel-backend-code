package observability

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/response"
)

type ObservabilityHandler struct{ service ObservabilityService }

func NewObservabilityHandler(c *di.Container) *ObservabilityHandler {
	return &ObservabilityHandler{service: NewObservabilityService(NewObservabilityRepository(), c.DB, c.Config)}
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

func (h *ObservabilityHandler) AskTLS(c fiber.Ctx) error {
	allowed, err := h.service.AllowTLS(c.Context(), c.Query("domain"))
	if err != nil || !allowed {
		return c.SendStatus(fiber.StatusForbidden)
	}
	return c.SendStatus(fiber.StatusOK)
}
